package service

import (
	"fmt"
	"strings"

	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
)

func NewDeadmessagePSM() (*dante_pb.DeadmessagePSM, error) {
	sm, err := dante_pb.DeadmessagePSMBuilder().BuildStateMachine()
	if err != nil {
		return nil, err
	}

	// new message
	sm.From(0).Mutate(dante_pb.DeadmessagePSMMutation(
		func(state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Notified) error {

			version := &dante_pb.DeadMessageVersion{
				VersionId: event.Notification.DeathId,
				Message:   event.Notification.Message,
			}

			if event.Notification.Infra != nil {
				infra := event.Notification.Infra
				switch infra.Type {
				case "SQS":
					sqsmsg := &dante_pb.DeadMessageVersion_SQSMessage{
						Attributes: make(map[string]string),
					}
					for k, v := range infra.Metadata {
						if k == "queueUrl" {
							sqsmsg.QueueUrl = v
						} else if strings.HasPrefix(k, "attr:") {
							k = strings.TrimPrefix(k, "attr:")
							sqsmsg.Attributes[k] = v
						}
					}
					version.SqsMessage = sqsmsg
				}
			}

			state.CurrentVersion = version
			state.Notification = event.Notification
			return nil
		})).
		SetStatus(dante_pb.MessageStatus_CREATED)

	// created to rejected
	sm.From(
		dante_pb.MessageStatus_CREATED,
		dante_pb.MessageStatus_REJECTED,
	).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Rejected) error {
			// how do we store the reason?

			return nil
		})).SetStatus(dante_pb.MessageStatus_REJECTED)

	// created to updated
	sm.From(
		dante_pb.MessageStatus_CREATED,
		dante_pb.MessageStatus_UPDATED,
	).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Updated) error {

			previous := state.CurrentVersion
			spec := event.Spec

			if spec.Message == nil {
				spec.Message = previous.Message
			}
			if spec.SqsMessage == nil {
				spec.SqsMessage = previous.SqsMessage
			} else if previous.SqsMessage != nil {

				if spec.SqsMessage.QueueUrl == "" {
					spec.SqsMessage.QueueUrl = previous.SqsMessage.QueueUrl
				}
				if spec.SqsMessage.Attributes == nil {
					spec.SqsMessage.Attributes = previous.SqsMessage.Attributes
				}
			}
			state.CurrentVersion = spec

			return nil
		})).SetStatus(dante_pb.MessageStatus_UPDATED)

	// [CREATED,UPDATED] -> [REPLAYED]
	sm.From(
		dante_pb.MessageStatus_CREATED,
		dante_pb.MessageStatus_UPDATED,
	).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Replayed) error {
			if state.CurrentVersion == nil {
				return fmt.Errorf("no message to replay")
			}
			if state.CurrentVersion.SqsMessage == nil || state.CurrentVersion.SqsMessage.QueueUrl == "" {
				return fmt.Errorf("no SQS message to replay")
			}

			return nil
		})).SetStatus(dante_pb.MessageStatus_REPLAYED)

	return sm, nil
}
