package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sq "github.com/elgris/sqrl"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/pentops/dante/dynamictype"
	"github.com/pentops/o5-go/dante/v1/dante_pb"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
	"github.com/pentops/protostate/psm"

	"github.com/pentops/o5-go/auth/v1/auth_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	protojson ProtoJSON
	sqsClient SqsSender
	slackUrl  string

	sm              *dante_pb.DeadmessagePSM
	messageQuerySet dante_spb.MessagePSMQuerySet

	dante_tpb.UnimplementedDeadMessageTopicServer
	dante_spb.UnimplementedDeadMessageQueryServiceServer
	dante_spb.UnimplementedDeadMessageCommandServiceServer
}

func (ds *DeadletterService) UpdateDeadMessage(ctx context.Context, req *dante_spb.UpdateDeadMessageRequest) (*dante_spb.UpdateDeadMessageResponse, error) {
	res := &dante_spb.UpdateDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
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

		newState, err := ds.sm.TransitionInTx(ctx, tx, &event)
		if err != nil {
			log.Infof(ctx, "update PSM error: %v", err.Error())
			return err
		}
		res.Message = newState

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter change to database")
		return nil, err
	}

	return res, nil
}

func (ds *DeadletterService) ReplayDeadMessage(ctx context.Context, req *dante_spb.ReplayDeadMessageRequest) (*dante_spb.ReplayDeadMessageResponse, error) {
	res := dante_spb.ReplayDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {
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
		s, err := ds.sm.TransitionInTx(ctx, tx, &event)
		if err != nil {
			log.Infof(ctx, "state machine transition error: %v", err.Error())
			return err
		}
		res.Message = s

		log.Infof(ctx, "s currentspec is %+v, grpc-message is '%v'", s.CurrentSpec, s.CurrentSpec.Payload.Proto.MessageName())

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
	res := dante_spb.RejectDeadMessageResponse{}

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

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

		s, err := ds.sm.TransitionInTx(ctx, tx, &event)
		if err != nil {
			log.Infof(ctx, "state machine transition error: %v", err.Error())
			return err
		}
		res.Message = s

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter change to database")
		return nil, err
	}

	return &res, nil
}

func (ds *DeadletterService) ListDeadMessageEvents(ctx context.Context, req *dante_spb.ListDeadMessageEventsRequest) (*dante_spb.ListDeadMessageEventsResponse, error) {
	res := &dante_spb.ListDeadMessageEventsResponse{}

	return res, ds.messageQuerySet.ListEvents(ctx, ds.db, req, res)
}

func (ds *DeadletterService) GetDeadMessage(ctx context.Context, req *dante_spb.GetDeadMessageRequest) (*dante_spb.GetDeadMessageResponse, error) {
	res := &dante_spb.GetDeadMessageResponse{}

	return res, ds.messageQuerySet.Get(ctx, ds.db, req, res)
}

func newPsm() (*dante_pb.DeadmessagePSM, error) {
	config := dante_pb.DefaultDeadmessagePSMConfig()
	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	// new message
	sm.From(dante_pb.MessageStatus_UNSPECIFIED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Created) error {

			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_CREATED
			state.CurrentSpec = event.Spec
			return nil
		}))

	// created to rejected
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Rejected) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED
			// how do we store the reason?

			return nil
		}))

	// created to updated
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Updated) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED
			state.CurrentSpec = event.Spec

			return nil
		}))

	// created to replayed
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Replayed) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REPLAYED

			return nil
		}))

	// updated to updated
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Updated) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED
			state.CurrentSpec = event.Spec

			return nil
		}))

	// updated to rejected
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Rejected) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED
			// how do we store the reason?

			return nil
		}))

	// updated to replayed
	sm.From(dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED).Do(
		dante_pb.DeadmessagePSMFunc(func(ctx context.Context,
			tb dante_pb.DeadmessagePSMTransitionBaton,
			state *dante_pb.DeadMessageState,
			event *dante_pb.DeadMessageEventType_Replayed) error {
			state.Status = dante_pb.MessageStatus_MESSAGE_STATUS_REPLAYED

			return nil
		}))

	return sm, nil
}

func (ds *DeadletterService) ListDeadMessages(ctx context.Context, req *dante_spb.ListDeadMessagesRequest) (*dante_spb.ListDeadMessagesResponse, error) {
	res := &dante_spb.ListDeadMessagesResponse{}

	return res, ds.messageQuerySet.List(ctx, ds.db, req, res)
}

type SlackMessage struct {
	Text string `json:"text"`
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

	_, err := ds.protojson.Marshal(&dms)
	if err != nil {
		log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
		// rely on the JSON version of the message
		dms.CurrentSpec.Payload.Proto = nil

		_, err = ds.protojson.Marshal(&dms)
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

	if err := ds.db.Transact(ctx, &sqrlx.TxOptions{
		Isolation: sql.LevelReadCommitted,
		ReadOnly:  false,
		Retryable: true,
	}, func(ctx context.Context, tx sqrlx.Transaction) error {

		_, err := ds.sm.TransitionInTx(ctx, tx, &event)
		if err != nil {
			log.Infof(ctx, "create PSM error: %v", err.Error())
			return err
		}

		return nil
	}); err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter to database")
		return nil, err
	}

	// if we got here, no error occurred so we inserted a new dead letter, let slack know
	if len(ds.slackUrl) > 0 {
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

func NewDeadletterServiceService(conn sqrlx.Connection, resolver dynamictype.Resolver, sqsClient SqsSender, slack string) (*DeadletterService, error) {
	db, err := sqrlx.New(conn, sq.Dollar)
	if err != nil {
		return nil, err
	}

	q, err := newPsm()
	if err != nil {
		return nil, fmt.Errorf("couldn't make new PSM: %w", err)
	}

	b := dante_spb.DefaultMessagePSMQuerySpec(q.StateTableSpec())

	qset, err := dante_spb.NewMessagePSMQuerySet(b, psm.StateQueryOptions{})
	if err != nil {
		return nil, fmt.Errorf("couldn't make new PSM query set: %w", err)
	}

	return &DeadletterService{
		db:              db,
		protojson:       dynamictype.NewProtoJSON(resolver),
		sqsClient:       sqsClient,
		slackUrl:        slack,
		messageQuerySet: *qset,
		sm:              q,
	}, nil

}
