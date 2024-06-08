package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pentops/dante/dynamictype"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
	"github.com/pentops/log.go/log"
	"github.com/pentops/o5-go/messaging/v1/messaging_tpb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/emptypb"
)

type DeadLetterWorker struct {
	db        *sqrlx.Wrapper
	sm        *dante_pb.DeadmessagePSM
	slackUrl  string
	protojson ProtoJSON

	messaging_tpb.UnimplementedDeadMessageTopicServer
}

func NewDeadLetterWorker(conn sqrlx.Connection, resolver dynamictype.Resolver, stateMachine *dante_pb.DeadmessagePSM, slack string) (*DeadLetterWorker, error) {
	db, err := sqrlx.New(conn, sqrlx.Dollar)
	if err != nil {
		return nil, err
	}

	return &DeadLetterWorker{
		db:        db,
		sm:        stateMachine,
		slackUrl:  slack,
		protojson: dynamictype.NewProtoJSON(resolver),
	}, nil

}

type SlackMessage struct {
	Text string `json:"text"`
}

func (ds *DeadLetterWorker) Dead(ctx context.Context, req *messaging_tpb.DeadMessage) (*emptypb.Empty, error) {

	notification := &dante_pb.DeadMessageNotification{
		DeathId:    req.DeathId,
		HandlerApp: req.HandlerApp,
		HandlerEnv: req.HandlerEnv,
		Message:    req.Message,
	}

	switch pr := req.Problem.Type.(type) {
	case *messaging_tpb.Problem_UnhandledError_:
		notification.Problem = &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: pr.UnhandledError.Error,
				},
			},
		}

	default:
		log.WithField(ctx, "problem", req.Problem).Error("Unknown problem type")
		notification.Problem = &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: "UNKNOWN PROBLEM TYPE",
				},
			},
		}
	}

	event := &dante_pb.DeadmessagePSMEventSpec{
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "Dante",
					EventName:  "Dead",
					ExternalId: &req.DeathId,
				},
			},
		},
		Keys: &dante_pb.DeadMessageKeys{
			MessageId: req.DeathId,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Event: &dante_pb.DeadMessageEventType_Created{
			Notification: notification,
		},
	}

	_, err := ds.sm.Transition(ctx, ds.db, event)
	if err != nil {
		log.WithError(ctx, err).Error("Couldn't save dead letter to database")
		return nil, err
	}

	// if we got here, no error occurred so we inserted a new dead letter, let slack know
	if len(ds.slackUrl) > 0 {
		msg := SlackMessage{}

		msg.Text = fmt.Sprintf("*Deadletter on \nEnv: %s App %s\nMethod /%s/%s\n*Error*:\n%s",
			req.HandlerEnv,
			req.HandlerApp,
			req.Message.GrpcService,
			req.Message.GrpcMethod,
			req.Problem.String(),
		)
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
