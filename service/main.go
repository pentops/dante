package service

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/pentops/o5-runtime-sidecar/awsmsg"
	"github.com/pentops/protostate/psm"

	"github.com/pentops/log.go/log"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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
	messageQuerySet dante_spb.DeadmessagePSMQuerySet

	dante_spb.UnimplementedDeadMessageQueryServiceServer
	dante_spb.UnimplementedDeadMessageCommandServiceServer
}

func NewDeadletterServiceService(conn sqrlx.Connection, statemachine *dante_pb.DeadmessagePSM, sqsClient SqsSender) (*DeadletterService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	qset, err := dante_spb.NewDeadmessagePSMQuerySet(
		dante_spb.DefaultDeadmessagePSMQuerySpec(statemachine.StateTableSpec()),
		psm.StateQueryOptions{},
	)
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

		// TODO: The remainder of this belongs in a hook -> outbox -> worker

		s := newState.Data

		log.Infof(ctx, "s currentspec is %+v, grpc-service is '/%s/%s'", s.CurrentVersion, s.CurrentVersion.Message.GrpcService, s.CurrentVersion.Message.GrpcMethod)

		messageBody, err := protojson.Marshal(s.CurrentVersion.Message)
		if err != nil {
			log.Errorf(ctx, "couldn't marshal message body: %v", err.Error())
			return fmt.Errorf("couldn't marshal message body: %w", err)
		}

		eventBridgeWrapper := &awsmsg.EventBridgeWrapper{
			Detail:     messageBody,
			DetailType: awsmsg.EventBridgeO5MessageDetailType,
		}

		jsonData, err := json.Marshal(eventBridgeWrapper)
		if err != nil {
			log.Errorf(ctx, "couldn't marshal eventbridge wrapper: %v", err.Error())
		}

		attributes := map[string]types.MessageAttributeValue{}
		for k, v := range s.CurrentVersion.SqsMessage.Attributes {
			attributes[k] = types.MessageAttributeValue{
				DataType:    aws.String("String"),
				StringValue: aws.String(v),
			}
		}

		i := sqs.SendMessageInput{
			MessageBody:       aws.String(string(jsonData)),
			QueueUrl:          aws.String(s.CurrentVersion.SqsMessage.QueueUrl),
			MessageAttributes: attributes,
		}
		log.Infof(ctx, "SQS message to be sent: %+v with body of %v", i, *i.MessageBody)
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
