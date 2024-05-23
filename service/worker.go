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
	"github.com/pentops/dante/gen/o5/dante/v1/dante_tpb"
	"github.com/pentops/protostate/gen/state/v1/psm_pb"
	"github.com/pentops/sqrlx.go/sqrlx"
	"google.golang.org/protobuf/types/known/emptypb"
	"gopkg.daemonl.com/log"
)

type DeadLetterWorker struct {
	db        *sqrlx.Wrapper
	sm        *dante_pb.DeadmessagePSM
	slackUrl  string
	protojson ProtoJSON

	dante_tpb.UnimplementedDeadMessageTopicServer
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

func (ds *DeadLetterWorker) Dead(ctx context.Context, req *dante_tpb.DeadMessage) (*emptypb.Empty, error) {
	s := dante_pb.DeadMessageSpec{
		VersionId:      uuid.NewString(),
		InfraMessageId: req.InfraMessageId,
		QueueName:      req.QueueName,
		GrpcName:       req.GrpcName,
		CreatedAt:      req.Timestamp,
		Payload:        req.Payload,
		Problem:        req.GetProblem(),
	}

	// While Dante has access to the proto definitions needed to convert the proto to json,
	// the protostate machine does not and will throw an error when trying to convert to json
	// to save to the database.
	// Dump the proto portion of the payload.
	s.Payload.Proto = nil

	_, err := ds.protojson.Marshal(&s)
	if err != nil {
		log.Infof(ctx, "couldn't turn dead letter into json after removing payload: %v", err.Error())
		return nil, err
	}

	event := &dante_pb.DeadmessagePSMEventSpec{
		Cause: &psm_pb.Cause{
			Type: &psm_pb.Cause_ExternalEvent{
				ExternalEvent: &psm_pb.ExternalEventCause{
					SystemName: "Dante",
					EventName:  "Dead",
					ExternalId: &req.MessageId,
				},
			},
		},
		Keys: &dante_pb.DeadMessageKeys{
			MessageId: req.MessageId,
		},
		EventID:   uuid.NewString(),
		Timestamp: time.Now(),
		Event: &dante_pb.DeadMessageEventType_Created{
			Spec: &s,
		},
	}

	_, err = ds.sm.Transition(ctx, ds.db, event)
	if err != nil {
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
