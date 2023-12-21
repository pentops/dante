package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	sq "github.com/elgris/sqrl"
	_ "github.com/lib/pq"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/dante/v1/dante_pb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"

	_ "github.com/pentops/o5-go/deployer/v1/deployer_pb"
	_ "github.com/pentops/o5-go/deployer/v1/deployer_tpb"

	"github.com/pressly/goose"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
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
	dante_pb.UnimplementedDanteQueryServiceServer
}

func (ds *DeadletterService) ListMessages(ctx context.Context, req *dante_pb.ListMessagesRequest) (*dante_pb.ListMessagesResponse, error) {
	res := dante_pb.ListMessagesResponse{}

	q := sq.Select(
		"deadletter",
	).From("messages").Limit(10) // pagination to be done later

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
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

			if err := rows.Scan(
				&deadJson,
			); err != nil {
				return err
			}

			deadProto := dante_pb.DeadMessage{}
			err = protojson.Unmarshal([]byte(deadJson), &deadProto)
			if err != nil {
				log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
				return err
			}
			res.Messages = append(res.Messages, &dante_pb.CapturedMessage{Cause: &deadProto})
		}

		return nil
	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(ctx, err).Error("Couldn't read dead letters")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) Dead(ctx context.Context, req *dante_tpb.DeadMessage) (*emptypb.Empty, error) {
	msg_json, err := protojson.Marshal(req.Dead)
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
			"message_id": req.Dead.MessageId,
			"deadletter": msg_json,
		}).Suffix("ON CONFLICT(message_id) DO NOTHING"))
		if err != nil {
			return err
		}

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

func runServe(ctx context.Context) error {
	type envConfig struct {
		PublicPort int `env:"PUBLIC_PORT" default:"8080"`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
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
		dante_pb.RegisterDanteQueryServiceServer(grpcServer, deadletterService)

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
