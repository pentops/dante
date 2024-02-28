package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pentops/dante/dynamictype"
	"github.com/pentops/log.go/grpc_log"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/auth/v1/auth_pb"
	"github.com/pentops/o5-go/dante/v1/dante_pb"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"github.com/pressly/goose"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.daemonl.com/envconf"
)

var Version string
var sqsClient *sqs.Client

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

type ProtoJSON interface {
	Marshal(v proto.Message) ([]byte, error)
	Unmarshal(data []byte, v proto.Message) error
}

type DeadletterService struct {
	db        *sqrlx.Wrapper
	protojson ProtoJSON
	slackUrl  string

	dante_tpb.UnimplementedDeadMessageTopicServer
	dante_spb.UnimplementedDeadMessageQueryServiceServer
	dante_spb.UnimplementedDeadMessageCommandServiceServer
}

func (ds *DeadletterService) UpdateDeadMessage(ctx context.Context, req *dante_spb.UpdateDeadMessageRequest) (*dante_spb.UpdateDeadMessageResponse, error) {
	res := dante_spb.UpdateDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		var deadletter, message_id string
		q := sq.Select("message_id, deadletter").From("messages").Where("message_id = ?", req.MessageId)
		err := tx.QueryRow(ctx, q).Scan(&message_id, &deadletter)
		if err != nil {
			return err
		}

		deadProto := dante_pb.DeadMessageState{}
		err = ds.protojson.Unmarshal([]byte(deadletter), &deadProto)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
			return err
		}
		deadProto.Status = dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED
		deadProto.CurrentSpec = req.Message

		msg_json, err := ds.protojson.Marshal(&deadProto)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
			return err
		}

		u := sq.Update("messages").SetMap(map[string]interface{}{
			"deadletter": msg_json,
		}).Where("message_id = ?", req.MessageId)

		_, err = tx.Update(ctx, u)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't update dead letter")
			return err
		}

		event := dante_pb.DeadMessageEvent{
			Metadata: &dante_pb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
				Actor:     &auth_pb.Actor{},
			},
			MessageId: req.MessageId,
			Event: &dante_pb.DeadMessageEventType{
				Type: &dante_pb.DeadMessageEventType_Updated_{
					Updated: &dante_pb.DeadMessageEventType_Updated{
						Spec: req.Message,
					},
				},
			},
		}

		event_json, err := ds.protojson.Marshal(&event)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
			return err
		}

		_, err = tx.Insert(ctx, sq.Insert("message_events").SetMap(map[string]interface{}{
			"message_id": req.MessageId,
			"msg_event":  event_json,
			"id":         event.Metadata.EventId,
		}))
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter change to database")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) ReplayDeadMessage(ctx context.Context, req *dante_spb.ReplayDeadMessageRequest) (*dante_spb.ReplayDeadMessageResponse, error) {
	res := dante_spb.ReplayDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		var deadletter, message_id string
		var raw sql.NullString
		q := sq.Select("message_id, deadletter, raw_msg").From("messages").Where("message_id = ?", req.MessageId)
		err := tx.QueryRow(ctx, q).Scan(&message_id, &deadletter, &raw)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't query dead letter from db")
			return err
		}

		deadProto := dante_pb.DeadMessageState{}
		err = ds.protojson.Unmarshal([]byte(deadletter), &deadProto)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
			return err
		}
		deadProto.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REPLAYED

		if deadProto.CurrentSpec.Payload == nil {
			log.Infof(ctx, "spec payload is nil, using raw msg")
			if !raw.Valid {
				log.Info(ctx, "tried to use raw msg but it was null")
				return fmt.Errorf("failed to fall back to raw message as it was null")
			}
			danteAny := dante_pb.Any{
				Json: raw.String,
			}
			log.Info(ctx, "raw msg used")
			deadProto.CurrentSpec.Payload = &danteAny
			log.Infof(ctx, "danteany is %v", danteAny.Json)
		}

		msg_json, err := ds.protojson.Marshal(&deadProto)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
			return err
		}

		u := sq.Update("messages").SetMap(map[string]interface{}{
			"deadletter": msg_json,
		}).Where("message_id = ?", req.MessageId)

		_, err = tx.Update(ctx, u)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't update dead letter")
			return err
		}

		event := dante_pb.DeadMessageEvent{
			Metadata: &dante_pb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
				Actor:     &auth_pb.Actor{},
			},
			MessageId: req.MessageId,
			Event: &dante_pb.DeadMessageEventType{
				Type: &dante_pb.DeadMessageEventType_Replayed_{
					Replayed: &dante_pb.DeadMessageEventType_Replayed{},
				},
			},
		}

		event_json, err := ds.protojson.Marshal(&event)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
			return err
		}

		_, err = tx.Insert(ctx, sq.Insert("message_events").SetMap(map[string]interface{}{
			"message_id": req.MessageId,
			"msg_event":  event_json,
			"id":         event.Metadata.EventId,
		}))
		if err != nil {
			log.Infof(ctx, "couldn't save message event: %v", err.Error())
			return err
		}

		if deadProto.CurrentSpec == nil {
			log.Infof(ctx, "spec is nil")
			return fmt.Errorf("deadproto currentspec is nil and shouldn't be")
		}
		if deadProto.CurrentSpec.Payload == nil {
			log.Infof(ctx, "spec payload is nil")
			return fmt.Errorf("deadproto currentspec payload is nil and shouldn't be")
		}
		if deadProto.CurrentSpec.Payload.Proto == nil {
			log.Infof(ctx, "spec payload proto is nil")
			return fmt.Errorf("deadproto currentspec payload proto is nil and shouldn't be")
		}

		// Direct SQS publish: pull this out into a function
		hdrs := map[string]types.MessageAttributeValue{
			"grpc-service": {
				DataType:    aws.String("String"),
				StringValue: aws.String(deadProto.CurrentSpec.GrpcName),
			},
			"grpc-message": {
				DataType:    aws.String("String"),
				StringValue: aws.String(fmt.Sprintf("%v", deadProto.CurrentSpec.Payload.Proto.MessageName())),
			},
			"Content-Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("application/json"),
			},
		}

		i := sqs.SendMessageInput{
			MessageBody:       &deadProto.CurrentSpec.Payload.Json,
			QueueUrl:          &deadProto.CurrentSpec.QueueName,
			MessageAttributes: hdrs,
		}
		log.Infof(ctx, "SQS message to be sent: %+v with body of %v", i, *i.MessageBody)
		_, err = sqsClient.SendMessage(ctx, &i)
		if err != nil {
			log.Errorf(ctx, "couldn't send SQS message for replay: %v", err.Error())
			return fmt.Errorf("couldn't send SQS message for replay: %w", err)
		}

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter change to database")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) RejectDeadMessage(ctx context.Context, req *dante_spb.RejectDeadMessageRequest) (*dante_spb.RejectDeadMessageResponse, error) {
	res := dante_spb.RejectDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		var deadletter, message_id string
		q := sq.Select("message_id, deadletter").From("messages").Where("message_id = ?", req.MessageId)
		err := tx.QueryRow(ctx, q).Scan(&message_id, &deadletter)
		if err != nil {
			return err
		}

		deadProto := dante_pb.DeadMessageState{}
		err = ds.protojson.Unmarshal([]byte(deadletter), &deadProto)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
			return err
		}
		deadProto.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED

		msg_json, err := ds.protojson.Marshal(&deadProto)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
			return err
		}

		u := sq.Update("messages").SetMap(map[string]interface{}{
			"deadletter": msg_json,
		}).Where("message_id = ?", req.MessageId)

		_, err = tx.Update(ctx, u)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't update dead letter")
			return err
		}

		event := dante_pb.DeadMessageEvent{
			Metadata: &dante_pb.Metadata{
				EventId:   uuid.NewString(),
				Timestamp: timestamppb.Now(),
				Actor:     &auth_pb.Actor{},
			},
			MessageId: req.MessageId,
			Event: &dante_pb.DeadMessageEventType{
				Type: &dante_pb.DeadMessageEventType_Rejected_{
					Rejected: &dante_pb.DeadMessageEventType_Rejected{
						Reason: req.Reason,
					},
				},
			},
		}

		event_json, err := ds.protojson.Marshal(&event)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
			return err
		}

		_, err = tx.Insert(ctx, sq.Insert("message_events").SetMap(map[string]interface{}{
			"message_id": req.MessageId,
			"msg_event":  event_json,
			"id":         event.Metadata.EventId,
		}))
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter change to database")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) ListDeadMessageEvents(ctx context.Context, req *dante_spb.ListDeadMessageEventsRequest) (*dante_spb.ListDeadMessageEventsResponse, error) {
	res := dante_spb.ListDeadMessageEventsResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  true,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
		// Limit instead of paging for now
		e := sq.Select("id, message_id, tstamp, msg_event").From("message_events").Where("message_id = ?", req.MessageId).OrderBy("tstamp DESC").Limit(100)
		rows, err := tx.Select(ctx, e)

		if err != nil {
			log.WithError(ctx, err).Error("Couldn't query database")
			return err
		}
		defer rows.Close()

		for rows.Next() {
			id := ""
			msg_id := ""
			tstamp := ""
			event := ""

			if err := rows.Scan(
				&id, &msg_id, &tstamp, &event,
			); err != nil {
				return err
			}

			eproto := dante_pb.DeadMessageEvent{}
			err := ds.protojson.Unmarshal([]byte(event), &eproto)
			if err != nil {
				log.WithError(ctx, err).Error("Couldn't unmarshal dead letter event")
				return err
			}
			res.Events = append(res.Events, &eproto)
		}
		return nil

	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(ctx, err).Error("Couldn't get dead letter events")
		return nil, err
	} else if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "Dead letter events not found")
	}

	return &res, nil
}

func (ds *DeadletterService) GetDeadMessage(ctx context.Context, req *dante_spb.GetDeadMessageRequest) (*dante_spb.GetDeadMessageResponse, error) {
	res := dante_spb.GetDeadMessageResponse{
		Events: make([]*dante_pb.DeadMessageEvent, 0),
	}

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
		if err != nil {
			return err
		}

		// Limit instead of paging for now
		e := sq.Select("id, message_id, tstamp, msg_event").From("message_events").Where("message_id = ?", *req.MessageId).OrderBy("tstamp DESC").Limit(100)
		rows, err := tx.Select(ctx, e)

		if err != nil {
			log.WithError(ctx, err).Error("Couldn't query database")
			return err
		}
		defer rows.Close()

		for rows.Next() {
			id := ""
			msg_id := ""
			tstamp := ""
			event := ""

			if err := rows.Scan(
				&id, &msg_id, &tstamp, &event,
			); err != nil {
				return err
			}

			eproto := dante_pb.DeadMessageEvent{}
			err := ds.protojson.Unmarshal([]byte(event), &eproto)
			if err != nil {
				log.WithError(ctx, err).Error("Couldn't unmarshal dead letter event")
				return err
			}
			res.Events = append(res.Events, &eproto)
		}
		return nil

	}); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.WithError(ctx, err).Error("Couldn't get dead letter")
		return nil, err
	} else if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, status.Error(codes.NotFound, "Dead letter not found")
	}

	deadProto := dante_pb.DeadMessageState{}
	err := ds.protojson.Unmarshal([]byte(deadletter), &deadProto)
	if err != nil {
		log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
		return nil, err
	}

	res.Message = &deadProto

	return &res, nil
}

func (ds *DeadletterService) ListDeadMessages(ctx context.Context, req *dante_spb.ListDeadMessagesRequest) (*dante_spb.ListDeadMessagesResponse, error) {
	res := dante_spb.ListDeadMessagesResponse{}

	q := sq.Select(
		"message_id, deadletter",
	).From("messages").OrderBy("created_at DESC").Limit(100) // pagination to be done later

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
			err = ds.protojson.Unmarshal([]byte(deadJson), &deadProto)
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

type SlackMessage struct {
	Text string `json:"text"`
}

func (ds *DeadletterService) Dead(ctx context.Context, req *dante_tpb.DeadMessage) (*emptypb.Empty, error) {
	rowInserted := false
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

	raw := ""
	msg_json, err := ds.protojson.Marshal(&dms)
	if err != nil {
		log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
		raw = dms.CurrentSpec.Payload.Json
		// Anything that has an "any" field is suspect
		dms.CurrentSpec.Payload = nil
		msg_json, err = ds.protojson.Marshal(&dms)
		if err != nil {
			log.Infof(ctx, "couldn't turn dead letter into json after removing payload: %v", err.Error())
			return nil, err
		}

	}

	event := dante_pb.DeadMessageEvent{
		Metadata: &dante_pb.Metadata{
			EventId:   uuid.NewString(),
			Timestamp: timestamppb.Now(),
			Actor:     &auth_pb.Actor{},
		},
		MessageId: dms.MessageId,
		Event: &dante_pb.DeadMessageEventType{
			Type: &dante_pb.DeadMessageEventType_Created_{
				Created: &dante_pb.DeadMessageEventType_Created{
					Spec: &s,
				},
			},
		},
	}

	event_json, err := ds.protojson.Marshal(&event)
	if err != nil {
		log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
		return nil, err
	}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		q := sq.Insert("messages").SetMap(map[string]interface{}{
			"message_id": dms.MessageId,
			"deadletter": msg_json,
		}).Suffix("ON CONFLICT(message_id) DO NOTHING")

		if len(raw) > 0 {
			q = sq.Insert("messages").SetMap(map[string]interface{}{
				"message_id": dms.MessageId,
				"deadletter": msg_json,
				"raw_msg":    raw,
			}).Suffix("ON CONFLICT(message_id) DO NOTHING")
		}

		r, err := tx.Insert(ctx, q)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't insert message to database")
			return err
		}
		p, err := r.RowsAffected()
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't get count of rows affected")
			return err
		}
		if p > 0 {
			rowInserted = true
		}

		_, err = tx.Insert(ctx, sq.Insert("message_events").SetMap(map[string]interface{}{
			"message_id": dms.MessageId,
			"msg_event":  event_json,
			"id":         event.Metadata.EventId,
		}))
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter to database")
		return nil, err
	}

	if len(ds.slackUrl) > 0 && rowInserted {
		msg := SlackMessage{}
		wrapper := `*Deadletter on*:
%v
*Error*:
%v}`

		msg.Text = fmt.Sprintf(wrapper, req.QueueName, req.Problem.String())
		json, err := json.Marshal(msg)
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't convert dead letter to slack message")
			msg.Text = "(Dante error converting to slack message)"
		}
		res, err := http.Post(ds.slackUrl, "application/json", bytes.NewReader([]byte(json)))
		if err != nil {
			log.WithError(ctx, err).Error("Couldn't send deadletter notice to slack")
		}
		defer res.Body.Close()
	}

	return &emptypb.Empty{}, nil
}

func NewDeadletterServiceService(conn sqrlx.Connection, resolver dynamictype.Resolver, slack string) (*DeadletterService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	return &DeadletterService{
		db:        db,
		protojson: dynamictype.NewProtoJSON(resolver),
		slackUrl:  slack,
	}, nil
}

func runServe(ctx context.Context) error {
	type envConfig struct {
		PublicPort  int    `env:"PUBLIC_PORT" default:"8080"`
		ProtobufSrc string `env:"PROTOBUF_SRC" default:""`
		SlackUrl    string `env:"SLACK_URL" default:""`
	}
	cfg := envConfig{}
	if err := envconf.Parse(&cfg); err != nil {
		return err
	}

	awsConfig, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	sqsClient = sqs.NewFromConfig(awsConfig)

	types := dynamictype.NewTypeRegistry()
	err = types.LoadExternalProtobufs(cfg.ProtobufSrc)
	if err != nil {
		return err
	}

	err = types.LoadProtoFromFile("./pentops-o5.binpb")
	if err != nil {
		return err
	}

	db, err := openDatabase(ctx)
	if err != nil {
		return err
	}

	deadletterService, err := NewDeadletterServiceService(db, types, cfg.SlackUrl)
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
