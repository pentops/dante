package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_spb"
	"github.com/pentops/protostate/psm"

	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"gopkg.daemonl.com/log"
)

type ProtoJSON interface {
	Marshal(v proto.Message) ([]byte, error)
	Unmarshal(data []byte, v proto.Message) error
}

type SqsSender interface {
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

type DeadletterService struct {
	db        *sqrlx.Wrapper
	sqsClient SqsSender

	sm              *dante_pb.DeadmessagePSM
	messageQuerySet dante_spb.MessagePSMQuerySet

	dante_spb.UnimplementedDeadMessageQueryServiceServer
	dante_spb.UnimplementedDeadMessageCommandServiceServer
}

func NewDeadletterServiceService(conn sqrlx.Connection, statemachine *dante_pb.DeadmessagePSM, sqsClient SqsSender) (*DeadletterService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	b := dante_spb.DefaultMessagePSMQuerySpec(statemachine.StateTableSpec())

	qset, err := dante_spb.NewMessagePSMQuerySet(b, psm.StateQueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't make new PSM query set: %w", err)
	}

	return &DeadletterService{
		db:              db,
		sqsClient:       sqsClient,
		messageQuerySet: *qset,
		sm:              statemachine,
	}, nil

}

func (ds *DeadletterService) UpdateDeadMessage(ctx context.Context, req *dante_spb.UpdateDeadMessageRequest) (*dante_spb.UpdateDeadMessageResponse, error) {
	res := &dante_spb.UpdateDeadMessageResponse{}

	event := &dante_pb.DeadmessagePSMEventSpec{
		Cause:     CommandCause(ctx),
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Keys: &dante_pb.DeadMessageKeys{
			MessageId: req.MessageId,
		},
		Event: &dante_pb.DeadMessageEventType_Updated{
			Spec: req.Message,
		},
	}

	newState, err := ds.sm.Transition(ctx, ds.db, event)
	if err != nil {
		log.Infof(ctx, "update PSM error: %v", err.Error())
		return nil, err
	}
	res.Message = newState

	return res, nil
}

func (ds *DeadletterService) ReplayDeadMessage(ctx context.Context, req *dante_spb.ReplayDeadMessageRequest) (*dante_spb.ReplayDeadMessageResponse, error) {
	res := dante_spb.ReplayDeadMessageResponse{}
	event := &dante_pb.DeadmessagePSMEventSpec{
		Cause: CommandCause(ctx),
		Keys: &dante_pb.DeadMessageKeys{
			MessageId: req.MessageId,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Event:     &dante_pb.DeadMessageEventType_Replayed{},
	}
	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		newState, err := ds.sm.TransitionInTx(ctx, tx, event)
		if err != nil {
			log.Infof(ctx, "state machine transition error: %v", err.Error())
			return err
		}
		res.Message = newState

		// TODO: This belongs in a hook.
		s := newState.Data

		log.Infof(ctx, "s currentspec is %+v, grpc-service is '%v'", s.CurrentSpec, s.CurrentSpec.GrpcName)

		// Direct SQS publish: pull this out into a function
		hdrs := map[string]types.MessageAttributeValue{
			"grpc-service": {
				DataType:    aws.String("String"),
				StringValue: aws.String(s.CurrentSpec.GrpcName),
			},
			"Content-Type": {
				DataType:    aws.String("String"),
				StringValue: aws.String("application/json"),
			},
		}

		i := sqs.SendMessageInput{
			MessageBody:       &s.CurrentSpec.Payload.Json,
			QueueUrl:          &s.CurrentSpec.QueueName,
			MessageAttributes: hdrs,
		}
		log.Infof(ctx, "SQS message to be sent: %+v with body of %v and headers %+v", i, *i.MessageBody, hdrs)
		_, err = ds.sqsClient.SendMessage(ctx, &i)
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
	res := &dante_spb.RejectDeadMessageResponse{}

	event := &dante_pb.DeadmessagePSMEventSpec{
		Cause:     CommandCause(ctx),
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Keys: &dante_pb.DeadMessageKeys{
			MessageId: req.MessageId,
		},
		Event: &dante_pb.DeadMessageEventType_Rejected{
			Reason: req.Reason,
		},
	}

	newState, err := ds.sm.Transition(ctx, ds.db, event)
	if err != nil {
		log.Infof(ctx, "update PSM error: %v", err.Error())
		return nil, err
	}
	res.Message = newState

	return res, nil
}

func (ds *DeadletterService) ListDeadMessageEvents(ctx context.Context, req *dante_spb.ListDeadMessageEventsRequest) (*dante_spb.ListDeadMessageEventsResponse, error) {
	res := &dante_spb.ListDeadMessageEventsResponse{}

	return res, ds.messageQuerySet.ListEvents(ctx, ds.db, req, res)
}

func (ds *DeadletterService) GetDeadMessage(ctx context.Context, req *dante_spb.GetDeadMessageRequest) (*dante_spb.GetDeadMessageResponse, error) {
	res := &dante_spb.GetDeadMessageResponse{}

	return res, ds.messageQuerySet.Get(ctx, ds.db, req, res)
}

func (ds *DeadletterService) ListDeadMessages(ctx context.Context, req *dante_spb.ListDeadMessagesRequest) (*dante_spb.ListDeadMessagesResponse, error) {
	res := &dante_spb.ListDeadMessagesResponse{}

	return res, ds.messageQuerySet.List(ctx, ds.db, req, res)
}
