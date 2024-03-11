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
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

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

	uu.Step("Create a dead message", func(t flowtest.Asserter) {
		// nil here at deadmessageworker
		_, err := uu.DeadMessageWorker.Dead(ctx, msg)

		t.NoError(err)
	})

	uu.Step("Get dead message", func(t flowtest.Asserter) {
		req := &dante_spb.GetDeadMessageRequest{
			MessageId: &msg.MessageId,
		}

		// listdeadmessages uses the psm for db access and unmarshalling, switch to that
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
	})
}
