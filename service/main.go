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

type DeadletterService struct {
	db        *sqrlx.Wrapper
	protojson ProtoJSON
	sqsClient *sqs.Client
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
		// var deadletter, message_id string
		// q := sq.Select("message_id, deadletter").From("deadmessage").Where("message_id = ?", req.MessageId)
		// err := tx.QueryRow(ctx, q).Scan(&message_id, &deadletter)
		// if err != nil {
		// 	return err
		// }

		// deadProto := dante_pb.DeadMessageState{}
		// err = ds.protojson.Unmarshal([]byte(deadletter), &deadProto)
		// if err != nil {
		// 	log.WithError(ctx, err).Error("Couldn't unmarshal dead letter")
		// 	return err
		// }
		// deadProto.Status = dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED
		// deadProto.CurrentSpec = req.Message

		// msg_json, err := ds.protojson.Marshal(&deadProto)
		// if err != nil {
		// 	log.Infof(ctx, "couldn't turn dead letter into json: %v", err.Error())
		// 	return err
		// }

		// u := sq.Update("deadmessage").SetMap(map[string]interface{}{
		// 	"deadletter": msg_json,
		// }).Where("message_id = ?", req.MessageId)

		// _, err = tx.Update(ctx, u)
		// if err != nil {
		// 	log.WithError(ctx, err).Error("Couldn't update dead letter")
		// 	return err
		// }

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

		// event_json, err := ds.protojson.Marshal(&event)
		// if err != nil {
		// 	log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
		// 	return err
		// }

		// _, err = tx.Insert(ctx, sq.Insert("deadmessage_event").SetMap(map[string]interface{}{
		// 	"message_id": req.MessageId,
		// 	"msg_event":  event_json,
		// 	"id":         event.Metadata.EventId,
		// }))
		// if err != nil {
		// 	return err
		// }

		_, err := ds.sm.TransitionInTx(ctx, tx, &event)
		if err != nil {
			log.Infof(ctx, "update PSM error: %v", err.Error())
			return err
		}

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
		// switch to PSM
		var deadletter, message_id string
		var raw sql.NullString
		q := sq.Select("message_id, deadletter, raw_msg").From("deadmessage").Where("message_id = ?", req.MessageId)
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

		u := sq.Update("deadmessage").SetMap(map[string]interface{}{
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

		_, err = tx.Insert(ctx, sq.Insert("deadmessage_event").SetMap(map[string]interface{}{
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

		// do we need to save this ourselves?
		// event_json, err := ds.protojson.Marshal(&event)
		// if err != nil {
		// 	log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
		// 	return err
		// }

		// _, err = tx.Insert(ctx, sq.Insert("message_events").SetMap(map[string]interface{}{
		// 	"message_id": req.MessageId,
		// 	"msg_event":  event_json,
		// 	"id":         event.Metadata.EventId,
		// }))
		// if err != nil {
		// 	return err
		// }

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

	// event_json, err := ds.protojson.Marshal(&event)
	// if err != nil {
	// 	log.Infof(ctx, "couldn't turn dead letter event into json: %v", err.Error())
	// 	return nil, err
	// }

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

		// q := sq.Insert("deadmessage").SetMap(map[string]interface{}{
		// 	"message_id": dms.MessageId,
		// 	"deadletter": msg_json,
		// }).Suffix("ON CONFLICT(message_id) DO NOTHING")

		// r, err := tx.Insert(ctx, q)
		// if err != nil {
		// 	log.WithError(ctx, err).Error("Couldn't insert message to database")
		// 	return err
		// }
		// p, err := r.RowsAffected()
		// if err != nil {
		// 	log.WithError(ctx, err).Error("Couldn't get count of rows affected")
		// 	return err
		// }
		// if p > 0 {
		// 	rowInserted = true
		// }

		// _, err = tx.Insert(ctx, sq.Insert("deadmessage_event").SetMap(map[string]interface{}{
		// 	"message_id": dms.MessageId,
		// 	"msg_event":  event_json,
		// 	"id":         event.Metadata.EventId,
		// }))
		// if err != nil {
		// 	return err
		// }

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

func NewDeadletterServiceService(conn sqrlx.Connection, resolver dynamictype.Resolver, sqsClient *sqs.Client, slack string) (*DeadletterService, error) {
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
