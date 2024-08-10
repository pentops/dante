package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
	"github.com/pentops/dante/gen/o5/dante/v1/dante_spb"
	"github.com/pentops/flowtest"
	"github.com/pentops/flowtest/prototest"
	"github.com/pentops/j5/gen/j5/list/v1/list_j5pb"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_pb"
	"github.com/pentops/o5-messaging/gen/o5/messaging/v1/messaging_tpb"
	"github.com/pentops/o5-runtime-sidecar/awsmsg"
	"github.com/pentops/realms/authtest"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestReplay(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &messaging_tpb.DeadMessage{
		DeathId:    uuid.NewString(),
		HandlerApp: "app",
		HandlerEnv: "env",
		Message: &messaging_pb.Message{
			MessageId:   uuid.NewString(),
			GrpcService: "test.Foo",
			GrpcMethod:  "Bar",
			Body: &messaging_pb.Any{
				TypeUrl:  "type.googleapis.com/test.Foo",
				Value:    []byte(`{"fake": "json"}`),
				Encoding: messaging_pb.WireEncoding_PROTOJSON,
			},
		},
		Infra: &messaging_tpb.Infra{
			Type: "SQS",
			Metadata: map[string]string{
				"queueUrl": "https://test/Queue",
				"ignore":   "me",
				"attr:foo": "bar",
			},
		},

		Problem: &messaging_tpb.Problem{
			Type: &messaging_tpb.Problem_UnhandledError_{
				UnhandledError: &messaging_tpb.Problem_UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(ctx context.Context, t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	// replay it
	uu.Step("Replay dead letter", func(ctx context.Context, t flowtest.Asserter) {
		ctx = authtest.JWTContext(ctx)
		msgsSent := len(uu.FakeSqs.Msgs)
		replayReq := &dante_spb.ReplayDeadMessageRequest{
			MessageId: msg.DeathId,
		}
		_, err := uu.DeadMessageCommand.ReplayDeadMessage(ctx, replayReq)
		t.NoError(err)

		t.Equal(msgsSent+1, len(uu.FakeSqs.Msgs))
		// assume latest message was ours
		a := uu.FakeSqs.Msgs[len(uu.FakeSqs.Msgs)-1]
		t.Equal("https://test/Queue", *a.QueueUrl)
		t.Equal("bar", *a.MessageAttributes["foo"].StringValue)
		_, hasIgnore := a.MessageAttributes["ignore"]
		t.Equal(false, hasIgnore)

		msg := decodeMessage(t, a.MessageBody)
		t.Equal("test.Foo", msg.GrpcService)
	})
}

type TB interface {
	Fatal(args ...interface{})
}

func decodeMessage(t TB, body *string) *messaging_pb.Message {
	if body == nil {
		t.Fatal("Body is nil")
		return nil // linter...
	}
	wrapper := awsmsg.EventBridgeWrapper{}
	if err := json.Unmarshal([]byte(*body), &wrapper); err != nil {
		t.Fatal(err)
	}

	msg := &messaging_pb.Message{}
	if err := protojson.Unmarshal(wrapper.Detail, msg); err != nil {
		t.Fatal(err)
	}

	return msg
}

func TestUpdate(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &messaging_tpb.DeadMessage{
		DeathId:    uuid.NewString(),
		HandlerApp: "app",
		HandlerEnv: "env",
		Message: &messaging_pb.Message{
			MessageId:   uuid.NewString(),
			GrpcService: "test.Foo",
			GrpcMethod:  "Bar",
			Body: &messaging_pb.Any{
				TypeUrl:  "type.googleapis.com/test.Foo",
				Value:    []byte(`{"fake": "json"}`),
				Encoding: messaging_pb.WireEncoding_PROTOJSON,
			},
		},

		Problem: &messaging_tpb.Problem{
			Type: &messaging_tpb.Problem_UnhandledError_{
				UnhandledError: &messaging_tpb.Problem_UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(ctx context.Context, t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("Get a specific dead message", func(ctx context.Context, t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.DeathId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)
		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED, resp.Message.Status)
	})

	uu.Step("Update the letter", func(ctx context.Context, t flowtest.Asserter) {
		ctx = authtest.JWTContext(ctx)
		newVersion := &dante_pb.DeadMessageVersion{
			VersionId: uuid.NewString(),
			Message: &messaging_pb.Message{
				MessageId:   uuid.NewString(),
				GrpcService: "test.Foo",
				GrpcMethod:  "NewMethod",
				Body: &messaging_pb.Any{
					TypeUrl:  "type.googleapis.com/test.Foo",
					Value:    []byte(`{"new": "json"}`),
					Encoding: messaging_pb.WireEncoding_PROTOJSON,
				},
			},
			SqsMessage: &dante_pb.DeadMessageVersion_SQSMessage{
				QueueUrl: "https://test/NewQueue",
			},
		}
		req := &dante_spb.UpdateDeadMessageRequest{
			MessageId:         msg.DeathId,
			ReplacesVersionId: msg.DeathId,
			Message:           newVersion,
		}

		resp, err := uu.DeadMessageCommand.UpdateDeadMessage(ctx, req)
		t.NoError(err)
		if resp == nil || resp.Message == nil {
			t.Fatal("Nothing in response")
		} else {
			if resp.Message != nil {
				t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED, resp.Message.Status)
			}
		}

		r := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.DeathId,
		}

		res, err := uu.DeadMessageQuery.GetDeadMessage(ctx, r)
		t.NoError(err)

		if res.Message == nil {
			t.Fatal("Message is nil")
		}

		m := res.Message
		t.Equal(msg.DeathId, m.Keys.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED)

		if len(res.Events) != 2 {
			t.Fatal("Too many events")
		}

		var updated *dante_pb.DeadMessageEventType_Updated
		var created *dante_pb.DeadMessageEventType_Notified
		for a := range res.Events {
			if res.Events[a].Event.GetNotified() != nil {
				created = res.Events[a].Event.GetNotified()
			}
			if res.Events[a].Event.GetUpdated() != nil {
				updated = res.Events[a].Event.GetUpdated()
			}
		}
		if created == nil {
			t.Fatal("Couldn't find created event")
		}
		if updated == nil {
			t.Fatal("Couldn't find updated event")
		} else { // a little silliness to convince the linter it's okay to access the variable
			t.Equal(`{"new": "json"}`, string(updated.Spec.Message.Body.Value))
		}
	})

	uu.Step("Replay dead letter", func(ctx context.Context, t flowtest.Asserter) {
		ctx = authtest.JWTContext(ctx)
		msgsSent := len(uu.FakeSqs.Msgs)
		replayReq := &dante_spb.ReplayDeadMessageRequest{
			MessageId: msg.DeathId,
		}
		_, err := uu.DeadMessageCommand.ReplayDeadMessage(ctx, replayReq)
		t.NoError(err)

		t.Equal(msgsSent+1, len(uu.FakeSqs.Msgs))
		// assume latest message was ours
		a := uu.FakeSqs.Msgs[len(uu.FakeSqs.Msgs)-1]
		msg := decodeMessage(t, a.MessageBody)
		t.Equal("test.Foo", msg.GrpcService)
		t.Equal("NewMethod", msg.GrpcMethod)
		t.Equal(`{"new": "json"}`, string(msg.Body.Value))
		t.Equal("https://test/NewQueue", *a.QueueUrl)
	})

}

func TestShelve(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &messaging_tpb.DeadMessage{
		DeathId:    uuid.NewString(),
		HandlerApp: "app",
		HandlerEnv: "env",
		Message: &messaging_pb.Message{
			MessageId:   uuid.NewString(),
			GrpcService: "test.Foo",
			GrpcMethod:  "Bar",
			Body: &messaging_pb.Any{
				TypeUrl:  "type.googleapis.com/test.Foo",
				Value:    []byte(`{"fake": "json"}`),
				Encoding: messaging_pb.WireEncoding_PROTOJSON,
			},
		},

		Problem: &messaging_tpb.Problem{
			Type: &messaging_tpb.Problem_UnhandledError_{
				UnhandledError: &messaging_tpb.Problem_UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(ctx context.Context, t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("Get a specific dead message", func(ctx context.Context, t flowtest.Asserter) {
		ctx = authtest.JWTContext(ctx)
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.DeathId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)
		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED, resp.Message.Status)
	})

	uu.Step("Shelve the letter", func(ctx context.Context, t flowtest.Asserter) {
		ctx = authtest.JWTContext(ctx)
		req := &dante_spb.RejectDeadMessageRequest{
			MessageId: msg.DeathId,
			Reason:    "not valid",
		}

		resp, err := uu.DeadMessageCommand.RejectDeadMessage(ctx, req)
		t.NoError(err)

		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED, resp.Message.Status)

		r := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.DeathId,
		}

		res, err := uu.DeadMessageQuery.GetDeadMessage(ctx, r)
		t.NoError(err)

		if res.Message == nil {
			t.Fatal("Message is nil")
		}

		m := res.Message
		t.Equal(msg.DeathId, m.Keys.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED)

		if len(res.Events) != 2 {
			t.Fatal("Too many events")
		}

		var rejected *dante_pb.DeadMessageEventType_Rejected
		var created *dante_pb.DeadMessageEventType_Notified
		for a := range res.Events {
			if res.Events[a].Event.GetNotified() != nil {
				created = res.Events[a].Event.GetNotified()
			}
			if res.Events[a].Event.GetRejected() != nil {
				rejected = res.Events[a].Event.GetRejected()
			}
		}
		if created == nil {
			t.Fatal("Couldn't find created event")
		}
		if rejected == nil {
			t.Fatal("Couldn't find rejected event")
		} else { // a little silliness to convince the linter it's okay to access the variable
			t.Equal("not valid", rejected.Reason)
		}
	})
}

func TestFieldPath(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	descFiles := prototest.DescriptorsFromSource(tt, map[string]string{
		"test.proto": `
			syntax = "proto3";

			package test;

			message Foo {
				string id = 1;
				string name = 2;
				int64 weight = 3;
			}
		`})

	fooDesc := descFiles.MessageByName(tt, "test.Foo")

	foo := dynamicpb.NewMessageType(fooDesc).New().Interface()

	b, err := protojson.Marshal(foo)
	if err != nil {
		tt.Fatal(err)
	}

	msg := &messaging_tpb.DeadMessage{
		DeathId:    uuid.NewString(),
		HandlerApp: "app",
		HandlerEnv: "env",
		Message: &messaging_pb.Message{
			MessageId:   uuid.NewString(),
			GrpcService: "test.Foo",
			GrpcMethod:  "Bar",
			Body: &messaging_pb.Any{
				TypeUrl:  "type.googleapis.com/test.Foo",
				Value:    b,
				Encoding: messaging_pb.WireEncoding_PROTOJSON,
			},
		},

		Problem: &messaging_tpb.Problem{
			Type: &messaging_tpb.Problem_UnhandledError_{
				UnhandledError: &messaging_tpb.Problem_UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create two dead letters", func(ctx context.Context, t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)

		msg.DeathId = uuid.NewString()
		_, err = uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("List all dead messages", func(ctx context.Context, t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 2 {
			t.Fatal("Should have exactly two dead letters")
		}
	})

	uu.Step("Get a small page of dead messages", func(ctx context.Context, t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{
			Page: &list_j5pb.PageRequest{
				PageSize: proto.Int64(1),
			},
		}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 1 {
			t.Fatal("Should have at least one dead letter")
		}
	})

	uu.Step("List sorted dead messages", func(ctx context.Context, t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{
			Query: &list_j5pb.QueryRequest{
				Sorts: []*list_j5pb.Sort{
					{Field: "metadata.createdAt"},
				},
			},
		}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 2 {
			t.Fatal("Should have exactly two dead letters")
		}
	})

	uu.Step("Get a specific dead message", func(ctx context.Context, t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.DeathId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)

		if resp.Message == nil {
			t.Fatal("Message is nil")
		}

		m := resp.Message

		t.Equal(msg.DeathId, m.Keys.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_CREATED)

		if m.Data.CurrentVersion == nil {
			t.Fatal("CurrentSpec is nil")
		}

		c := m.Data.CurrentVersion

		if c.Message == nil || c.Message.Body == nil {
			t.Fatal("Payload is nil")
		}

		t.Equal(1, len(resp.Events))
	})
}
