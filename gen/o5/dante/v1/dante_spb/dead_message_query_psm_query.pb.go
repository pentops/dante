// Code generated by protoc-gen-go-psm. DO NOT EDIT.

package dante_spb

import (
	psm "github.com/pentops/protostate/psm"
)

// State Query Service for %sDeadmessage
// QuerySet is the query set for the Deadmessage service.

type DeadmessagePSMQuerySet = psm.StateQuerySet[
	*GetDeadMessageRequest,
	*GetDeadMessageResponse,
	*ListDeadMessagesRequest,
	*ListDeadMessagesResponse,
	*ListDeadMessageEventsRequest,
	*ListDeadMessageEventsResponse,
]

func NewDeadmessagePSMQuerySet(
	smSpec psm.QuerySpec[
		*GetDeadMessageRequest,
		*GetDeadMessageResponse,
		*ListDeadMessagesRequest,
		*ListDeadMessagesResponse,
		*ListDeadMessageEventsRequest,
		*ListDeadMessageEventsResponse,
	],
	options psm.StateQueryOptions,
) (*DeadmessagePSMQuerySet, error) {
	return psm.BuildStateQuerySet[
		*GetDeadMessageRequest,
		*GetDeadMessageResponse,
		*ListDeadMessagesRequest,
		*ListDeadMessagesResponse,
		*ListDeadMessageEventsRequest,
		*ListDeadMessageEventsResponse,
	](smSpec, options)
}

type DeadmessagePSMQuerySpec = psm.QuerySpec[
	*GetDeadMessageRequest,
	*GetDeadMessageResponse,
	*ListDeadMessagesRequest,
	*ListDeadMessagesResponse,
	*ListDeadMessageEventsRequest,
	*ListDeadMessageEventsResponse,
]

func DefaultDeadmessagePSMQuerySpec(tableSpec psm.QueryTableSpec) DeadmessagePSMQuerySpec {
	return psm.QuerySpec[
		*GetDeadMessageRequest,
		*GetDeadMessageResponse,
		*ListDeadMessagesRequest,
		*ListDeadMessagesResponse,
		*ListDeadMessageEventsRequest,
		*ListDeadMessageEventsResponse,
	]{
		QueryTableSpec: tableSpec,
		ListRequestFilter: func(req *ListDeadMessagesRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			return filter, nil
		},
		ListEventsRequestFilter: func(req *ListDeadMessageEventsRequest) (map[string]interface{}, error) {
			filter := map[string]interface{}{}
			filter["message_id"] = req.MessageId
			return filter, nil
		},
	}
}
