package service

import (
	"github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
)

func NewDeadmessagePSM() (*dante_pb.DeadmessagePSM, error) {
	config := dante_pb.DefaultDeadmessagePSMConfig()
	sm, err := config.NewStateMachine()
	if err != nil {
		return nil, err
	}

	// new message
	sm.From(0).Mutate(dante_pb.DeadmessagePSMMutation(
		func(state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Created) error {
			state.CurrentSpec = event.Spec
			return nil
		})).
		SetStatus(dante_pb.MessageStatus_CREATED)

	// created to rejected
	sm.From(dante_pb.MessageStatus_CREATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Rejected) error {
			// how do we store the reason?

			return nil
		})).SetStatus(dante_pb.MessageStatus_REJECTED)

	// created to updated
	sm.From(dante_pb.MessageStatus_CREATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Updated) error {
			state.CurrentSpec = event.Spec

			return nil
		})).SetStatus(dante_pb.MessageStatus_UPDATED)

	// created to replayed
	sm.From(dante_pb.MessageStatus_CREATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Replayed) error {

			return nil
		})).SetStatus(dante_pb.MessageStatus_REPLAYED)

	// updated to updated
	sm.From(dante_pb.MessageStatus_UPDATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Updated) error {
			state.CurrentSpec = event.Spec

			return nil
		})).SetStatus(dante_pb.MessageStatus_UPDATED)

	// updated to rejected
	sm.From(dante_pb.MessageStatus_UPDATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Rejected) error {
			// how do we store the reason?

			return nil
		})).SetStatus(dante_pb.MessageStatus_REJECTED)

	// updated to replayed
	sm.From(dante_pb.MessageStatus_UPDATED).Mutate(
		dante_pb.DeadmessagePSMMutation(func(
			state *dante_pb.DeadMessageData,
			event *dante_pb.DeadMessageEventType_Replayed) error {

			return nil
		})).SetStatus(dante_pb.MessageStatus_REPLAYED)

	return sm, nil
}
