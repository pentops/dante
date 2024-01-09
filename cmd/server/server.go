package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/dante/v1/dante_pb"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
	"github.com/pressly/goose"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.daemonl.com/envconf"
	"gopkg.daemonl.com/sqrlx"
)

var Version string

func main() {
	ctx := context.Background()
	ctx = log.WithFields(ctx, map[string]interface{}{
		"application": "dante",
		"version":     Version,
	})

	args := os.Args[1:]
	if len(args) == 0 {
		args = append(args, "serve")
	}

	switch args[0] {
	case "serve":
		if err := runServe(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to serve")
			os.Exit(1)
		}

	case "migrate":
		if err := runMigrate(ctx); err != nil {
			log.WithError(ctx, err).Error("Failed to migrate")
			os.Exit(1)
		}

	default:
		log.WithField(ctx, "command", args[0]).Error("Unknown command")
		os.Exit(1)
	}
}

func runMigrate(ctx context.Context) error {
	var config = struct {
		MigrationsDir string `env:"MIGRATIONS_DIR" default:"./ext/db"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := openDatabase(ctx)
	if err != nil {
		return err
	}

	return goose.Up(db, "/migrations")
}

func openDatabase(ctx context.Context) (*sql.DB, error) {
	var config = struct {
		PostgresURL string `env:"POSTGRES_URL"`
	}{}

	if err := envconf.Parse(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	db, err := sql.Open("postgres", config.PostgresURL)
	if err != nil {
		return nil, err
	}

	for {
		if err := db.Ping(); err != nil {
			log.WithError(ctx, err).Error("pinging PG")
			time.Sleep(time.Second)
			continue
		}
		break
	}

	return db, nil
}

type DeadletterService struct {
	db *sqrlx.Wrapper

	dante_tpb.UnimplementedDeadMessageTopicServer
	dante_spb.UnimplementedDeadMessageQueryServiceServer
	dante_spb.UnimplementedDeadMessageCommandServiceServer
}

func (ds *DeadletterService) ListDeadMessageEvents(ctx context.Context, req *dante_spb.ListDeadMessageEventsRequest) (*dante_spb.ListDeadMessageEventsResponse, error) {

	return nil, nil
}

func (ds *DeadletterService) GetDeadMessage(ctx context.Context, req *dante_spb.GetDeadMessageRequest) (*dante_spb.GetDeadMessageResponse, error) {
	res := dante_spb.GetDeadMessageResponse{}

	var message_id string
	var created_at, updated_at string
	var deadletter string

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		q := sq.Select("message_id, created_at, updated_at, deadletter").From("messages").Where("message_id = ?", *req.MessageId)
		err := tx.QueryRow(ctx, q).Scan(&message_id, &created_at, &updated_at, &deadletter)
		return err

	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(ctx, err).Error("Couldn't get dead letter")
		return nil, err
	} else if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "Dead letter not found")
	}

	deadProto := dante_pb.DeadMessageState{}
	err := protojson.Unmarshal([]byte(deadletter), &deadProto)
	if err != nil {
		log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
		return nil, err
	}

	res.Message = &deadProto
	// TODO: this will be another db fetch
	res.Events = make([]*dante_pb.DeadMessageEvent, 0)

	return &res, nil
}

func (ds *DeadletterService) ListDeadMessages(ctx context.Context, req *dante_spb.ListDeadMessagesRequest) (*dante_spb.ListDeadMessagesResponse, error) {
	res := dante_spb.ListDeadMessagesResponse{}

	q := sq.Select(
		"message_id, deadletter",
	).From("messages").Limit(10) // pagination to be done later

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		rows, err := tx.Select(ctx, q)

		if err != nil {
			log.WithError(ctx, err).Error("Couldn't query database")
			return err
		}
		defer rows.Close()

		for rows.Next() {
			deadJson := ""
			msg_id := ""

			if err := rows.Scan(
				&msg_id, &deadJson,
			); err != nil {
				return err
			}

			deadProto := dante_pb.DeadMessageState{}
			err = protojson.Unmarshal([]byte(deadJson), &deadProto)
			if err != nil {
				log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
				return err
			}
			res.Messages = append(res.Messages, &deadProto)
		}

		return nil
	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(ctx, err).Error("Couldn't read dead letters")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) Dead(ctx context.Context, req *dante_tpb.DeadMessage) (*emptypb.Empty, error) {
	s := dante_pb.DeadMessageSpec{
		VersionId:      uuid.NewString(),
		InfraMessageId: req.InfraMessageId,
		QueueName:      req.QueueName,
		GrpcName:       req.GrpcName,
		CreatedAt:      req.Timestamp,
		Payload:        req.Payload,
		Problem:        req.GetProblem(),
	}

	dms := dante_pb.DeadMessageState{
		MessageId:   req.MessageId,
		CurrentSpec: &s,
		Status:      dante_pb.MessageStatus_CREATED,
	}

	msg_json, err := protojson.Marshal(&dms)
	if err != nil {
		log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
		return nil, err
	}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		_, err := tx.Insert(ctx, sq.Insert("messages").SetMap(map[string]interface{}{
			"message_id": dms.MessageId,
			"deadletter": msg_json,
		}).Suffix("ON CONFLICT(message_id) DO NOTHING"))
		if err != nil {
			return err
		}

		// do we also add an entry to message events?

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter to database")
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func NewDeadletterServiceService(conn sqrlx.Connection) (*DeadletterService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &DeadletterService{
		db: db,
	}, nil
}

func loadProtoFromFile(ctx context.Context, fileName string) error {
	protoFile, err := os.ReadFile(fileName)
	if err != nil {
		log.Infof(ctx, "Couldn't read file: %v", err.Error())
		return err
	}

	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(protoFile, fds)
	if err != nil {
		log.Infof(ctx, "Couldn't unmarshal file protofile: %v", err.Error())
		return err
	}
	fp, err := protodesc.NewFiles(fds)
	if err != nil {
		log.Infof(ctx, "Couldn't convert to registry file: %v", err.Error())
		return err
	}

	fp.RangeFiles(func(a protoreflect.FileDescriptor) bool {
		q, _ := protoregistry.GlobalFiles.FindFileByPath(a.Path())
		if q == nil {
			err := protoregistry.GlobalFiles.RegisterFile(a)
			if err != nil {
				log.Infof(ctx, "Couldn't load %v into protoregistry: %v", a, err.Error())
			}
		}
		return true
	})
	return nil
}

func loadLocalExternalProtobufs(ctx context.Context, fileName string) error {
	before := protoregistry.GlobalFiles.NumFiles()

	err := loadProtoFromFile(ctx, fileName)
	if err != nil {
		log.Infof(ctx, "Couldn't load file from local source: %v", err.Error())
	}

	loaded := protoregistry.GlobalFiles.NumFiles() - before
	if loaded > 0 {
		log.Infof(ctx, "Loaded %v external local proto defs", loaded)
	} else {
		log.Info(ctx, "Attempted to load external local proto defs but none were registered")
	}
	return nil
}

func loadExternalProtobufs(ctx context.Context, src string) error {
	if len(src) == 0 {
		return nil
	}

	res, err := http.Get(src)
	if err != nil {
		log.Infof(ctx, "Couldn't get file from %v: %v", src, err.Error())
		return err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode > 299 {
		log.Infof(ctx, "got status code %v instead of 2xx", res.StatusCode)
		return err
	}
	file := "./tmpprotos"

	fd, err := os.Create(file)
	if err != nil {
		log.Infof(ctx, "Couldn't create file to download to: %v", err.Error())
		return err
	}
	_, err = fd.Write(body)
	if err != nil {
		fd.Close()
		log.Infof(ctx, "Couldn't write file: %v", err.Error())
		return err
	}
	fd.Close()

	before := protoregistry.GlobalFiles.NumFiles()
	err = loadProtoFromFile(ctx, file)
	if err != nil {
		log.Infof(ctx, "Couldn't load file from http source: %v", err.Error())
	}

	loaded := protoregistry.GlobalFiles.NumFiles() - before
	if loaded > 0 {
		log.Infof(ctx, "Loaded %v external proto defs via http", loaded)
	} else {
		log.Info(ctx, "Attempted to load external proto defs but none were registered")
	}

	return nil
}

func runServe(ctx context.Context) error {
	type envConfig struct {
		PublicPort  int    `env:"PUBLIC_PORT" default:"8080"`
		ProtobufSrc string `env:"PROTOBUF_SRC" default:""`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
		return err
	}

	err := loadExternalProtobufs(ctx, cfg.ProtobufSrc)
	if err != nil {
		return err
	}

	err = loadLocalExternalProtobufs(ctx, "./pentops-o5.binpb")
	if err != nil {
		return err
	}

	db, err := openDatabase(ctx)
	if err != nil {
		return err
	}

	deadletterService, err := NewDeadletterServiceService(db)
	if err != nil {
		return err
	}

	log.WithField(ctx, "PORT", cfg.PublicPort).Info("Boot")

	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
			grpc_log.UnaryServerInterceptor(log.DefaultContext, log.DefaultTrace, log.DefaultLogger),
		))

		reflection.Register(grpcServer)

		dante_tpb.RegisterDeadMessageTopicServer(grpcServer, deadletterService)
		dante_spb.RegisterDeadMessageCommandServiceServer(grpcServer, deadletterService)
		dante_spb.RegisterDeadMessageQueryServiceServer(grpcServer, deadletterService)

		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.PublicPort))
		if err != nil {
			return err
		}
		log.WithField(ctx, "port", cfg.PublicPort).Info("Begin Worker Server")
		go func() {
			<-ctx.Done()
			grpcServer.GracefulStop() // nolint:errcheck
		}()

		return grpcServer.Serve(lis)
	})

	return eg.Wait()
}
