package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/pentops/flowtest"
	"github.com/pentops/jsonapi/prototest"
	"github.com/pentops/o5-go/dante/v1/dante_pb"
	"github.com/pentops/o5-go/dante/v1/dante_spb"
	"github.com/pentops/o5-go/dante/v1/dante_tpb"
	"github.com/pentops/protostate/gen/list/v1/psml_pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestReplay(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &dante_tpb.DeadMessage{
		MessageId:      uuid.NewString(),
		InfraMessageId: uuid.NewString(),

		QueueName: "test",
		GrpcName:  "test.Foo",

		Timestamp: timestamppb.Now(),

		Payload: &dante_pb.Any{
			Json: string("fake json"),
		},

		Problem: &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	// replay it
	uu.Step("Replay dead letter", func(t flowtest.Asserter) {
		msgsSent := len(uu.FakeSqs.Msgs)
		replayReq := &dante_spb.ReplayDeadMessageRequest{
			MessageId: msg.MessageId,
		}
		_, err := uu.DeadMessageCommand.ReplayDeadMessage(ctx, replayReq)
		t.NoError(err)

		t.Equal(msgsSent+1, len(uu.FakeSqs.Msgs))
		// assume latest message was ours
		a := uu.FakeSqs.Msgs[len(uu.FakeSqs.Msgs)-1]
		t.Equal("application/json", *a.MessageAttributes["Content-Type"].StringValue)
		t.Equal("test.Foo", *a.MessageAttributes["grpc-service"].StringValue)
	})

}

func TestUpdate(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &dante_tpb.DeadMessage{
		MessageId:      uuid.NewString(),
		InfraMessageId: uuid.NewString(),

		QueueName: "test",
		GrpcName:  "test.Foo",

		Timestamp: timestamppb.Now(),

		Payload: &dante_pb.Any{
			Json: string("fake json"),
		},

		Problem: &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("Get a specific dead message", func(t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.MessageId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)
		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED, resp.Message.Status)
	})

	uu.Step("Update the letter", func(t flowtest.Asserter) {
		newVersion := dante_pb.DeadMessageSpec{
			VersionId:      uuid.NewString(),
			InfraMessageId: msg.InfraMessageId,
			QueueName:      "new-queue-name",
			GrpcName:       msg.GrpcName,
			Payload:        msg.Payload,
			CreatedAt:      timestamppb.Now(),
		}
		req := &dante_spb.UpdateDeadMessageRequest{
			MessageId: msg.MessageId,
			Message:   &newVersion,
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
			MessageId: &msg.MessageId,
		}

		res, err := uu.DeadMessageQuery.GetDeadMessage(ctx, r)
		t.NoError(err)

		if res.Message == nil {
			t.Fatal("Message is nil")
		}

		m := res.Message
		t.Equal(msg.MessageId, m.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_MESSAGE_STATUS_UPDATED)

		if len(res.Events) != 2 {
			t.Fatal("Too many events")
		}

		var updated *dante_pb.DeadMessageEventType_Updated
		var created *dante_pb.DeadMessageEventType_Created
		for a := range res.Events {
			if res.Events[a].Event.GetCreated() != nil {
				created = res.Events[a].Event.GetCreated()
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
			t.Equal("new-queue-name", updated.Spec.QueueName)
		}
	})
}

func TestShelve(tt *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	uu := NewUniverse(ctx, tt)
	defer uu.RunSteps(tt)

	msg := &dante_tpb.DeadMessage{
		MessageId:      uuid.NewString(),
		InfraMessageId: uuid.NewString(),

		QueueName: "test",
		GrpcName:  "test.Foo",

		Timestamp: timestamppb.Now(),

		Payload: &dante_pb.Any{
			Json: string("fake json"),
		},

		Problem: &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create a dead letter", func(t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("Get a specific dead message", func(t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.MessageId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)
		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_CREATED, resp.Message.Status)
	})

	uu.Step("Shelve the letter", func(t flowtest.Asserter) {
		req := &dante_spb.RejectDeadMessageRequest{
			MessageId: msg.MessageId,
			Reason:    "not valid",
		}

		resp, err := uu.DeadMessageCommand.RejectDeadMessage(ctx, req)
		t.NoError(err)

		t.Equal(dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED, resp.Message.Status)

		r := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.MessageId,
		}

		res, err := uu.DeadMessageQuery.GetDeadMessage(ctx, r)
		t.NoError(err)

		if res.Message == nil {
			t.Fatal("Message is nil")
		}

		m := res.Message
		t.Equal(msg.MessageId, m.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_MESSAGE_STATUS_REJECTED)

		if len(res.Events) != 2 {
			t.Fatal("Too many events")
		}

		var rejected *dante_pb.DeadMessageEventType_Rejected
		var created *dante_pb.DeadMessageEventType_Created
		for a := range res.Events {
			if res.Events[a].Event.GetCreated() != nil {
				created = res.Events[a].Event.GetCreated()
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
	var msg *dante_tpb.DeadMessage

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

	pb, err := proto.Marshal(foo)
	if err != nil {
		tt.Fatal(err)
	}

	b, err := protojson.Marshal(foo)
	if err != nil {
		tt.Fatal(err)
	}

	msg = &dante_tpb.DeadMessage{
		MessageId:      uuid.NewString(),
		InfraMessageId: uuid.NewString(),

		QueueName: "test",
		GrpcName:  "test.Foo",

		Timestamp: timestamppb.Now(),

		Payload: &dante_pb.Any{
			Proto: &anypb.Any{
				TypeUrl: "type.googleapis.com/test.Foo",
				Value:   pb,
			},
			Json: string(b),
		},

		Problem: &dante_pb.Problem{
			Type: &dante_pb.Problem_UnhandledError{
				UnhandledError: &dante_pb.UnhandledError{
					Error: "test error",
				},
			},
		},
	}

	uu.Step("Create two dead letters", func(t flowtest.Asserter) {
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)

		msg.MessageId = uuid.NewString()
		_, err = uu.DeadMessageWorker.Dead(ctx, msg)
		t.NoError(err)
	})

	uu.Step("List all dead messages", func(t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 2 {
			t.Fatal("Should have exactly two dead letters")
		}
	})

	uu.Step("Get a small page of dead messages", func(t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{
			Page: &psml_pb.PageRequest{
				PageSize: proto.Int64(1),
			},
		}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 1 {
			t.Fatal("Should have at least one dead letter")
		}
	})

	uu.Step("List sorted dead messages", func(t flowtest.Asserter) {
		req := &dante_spb.ListDeadMessagesRequest{
			Query: &psml_pb.QueryRequest{
				Sorts: []*psml_pb.Sort{
					{Field: "currentSpec.createdAt"},
				},
			},
		}

		resp, err := uu.DeadMessageQuery.ListDeadMessages(ctx, req)
		t.NoError(err)
		if len(resp.Messages) != 2 {
			t.Fatal("Should have exactly two dead letters")
		}
	})

	uu.Step("Get a specific dead message", func(t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.MessageId,
		}

		resp, err := uu.DeadMessageQuery.GetDeadMessage(ctx, req)
		t.NoError(err)

		if resp.Message == nil {
			t.Fatal("Message is nil")
		}

		m := resp.Message

		t.Equal(msg.MessageId, m.MessageId)
		t.Equal(m.Status, dante_pb.MessageStatus_CREATED)

		if m.CurrentSpec == nil {
			t.Fatal("CurrentSpec is nil")
		}

		c := m.CurrentSpec

		t.Equal(msg.InfraMessageId, c.InfraMessageId)
		t.Equal(msg.QueueName, c.QueueName)

		if c.Payload == nil {
			t.Fatal("Payload is nil")
		}

		t.Equal(1, len(resp.Events))
	})
}
