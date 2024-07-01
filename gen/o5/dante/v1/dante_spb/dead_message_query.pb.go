// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        (unknown)
// source: o5/dante/v1/service/dead_message_query.proto

package dante_spb

import (
	_ "buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
	dante_pb "github.com/pentops/dante/gen/o5/dante/v1/dante_pb"
	_ "github.com/pentops/o5-auth/gen/o5/auth/v1/auth_pb"
	psml_pb "github.com/pentops/protostate/gen/list/v1/psml_pb"
	_ "github.com/pentops/protostate/gen/state/v1/psm_pb"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GetDeadMessageRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// when not set, returns the next unhandled message
	MessageId *string `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3,oneof" json:"message_id,omitempty"`
}

func (x *GetDeadMessageRequest) Reset() {
	*x = GetDeadMessageRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetDeadMessageRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetDeadMessageRequest) ProtoMessage() {}

func (x *GetDeadMessageRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetDeadMessageRequest.ProtoReflect.Descriptor instead.
func (*GetDeadMessageRequest) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{0}
}

func (x *GetDeadMessageRequest) GetMessageId() string {
	if x != nil && x.MessageId != nil {
		return *x.MessageId
	}
	return ""
}

type GetDeadMessageResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message *dante_pb.DeadMessageState   `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Events  []*dante_pb.DeadMessageEvent `protobuf:"bytes,2,rep,name=events,proto3" json:"events,omitempty"`
}

func (x *GetDeadMessageResponse) Reset() {
	*x = GetDeadMessageResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GetDeadMessageResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetDeadMessageResponse) ProtoMessage() {}

func (x *GetDeadMessageResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetDeadMessageResponse.ProtoReflect.Descriptor instead.
func (*GetDeadMessageResponse) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{1}
}

func (x *GetDeadMessageResponse) GetMessage() *dante_pb.DeadMessageState {
	if x != nil {
		return x.Message
	}
	return nil
}

func (x *GetDeadMessageResponse) GetEvents() []*dante_pb.DeadMessageEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

type ListDeadMessagesRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Page  *psml_pb.PageRequest  `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
	Query *psml_pb.QueryRequest `protobuf:"bytes,101,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListDeadMessagesRequest) Reset() {
	*x = ListDeadMessagesRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDeadMessagesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDeadMessagesRequest) ProtoMessage() {}

func (x *ListDeadMessagesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDeadMessagesRequest.ProtoReflect.Descriptor instead.
func (*ListDeadMessagesRequest) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{2}
}

func (x *ListDeadMessagesRequest) GetPage() *psml_pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListDeadMessagesRequest) GetQuery() *psml_pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListDeadMessagesResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Messages []*dante_pb.DeadMessageState `protobuf:"bytes,1,rep,name=messages,proto3" json:"messages,omitempty"`
	Page     *psml_pb.PageResponse        `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListDeadMessagesResponse) Reset() {
	*x = ListDeadMessagesResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDeadMessagesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDeadMessagesResponse) ProtoMessage() {}

func (x *ListDeadMessagesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDeadMessagesResponse.ProtoReflect.Descriptor instead.
func (*ListDeadMessagesResponse) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{3}
}

func (x *ListDeadMessagesResponse) GetMessages() []*dante_pb.DeadMessageState {
	if x != nil {
		return x.Messages
	}
	return nil
}

func (x *ListDeadMessagesResponse) GetPage() *psml_pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

type ListDeadMessageEventsRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MessageId string                `protobuf:"bytes,1,opt,name=message_id,json=messageId,proto3" json:"message_id,omitempty"`
	Page      *psml_pb.PageRequest  `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
	Query     *psml_pb.QueryRequest `protobuf:"bytes,101,opt,name=query,proto3" json:"query,omitempty"`
}

func (x *ListDeadMessageEventsRequest) Reset() {
	*x = ListDeadMessageEventsRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDeadMessageEventsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDeadMessageEventsRequest) ProtoMessage() {}

func (x *ListDeadMessageEventsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDeadMessageEventsRequest.ProtoReflect.Descriptor instead.
func (*ListDeadMessageEventsRequest) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{4}
}

func (x *ListDeadMessageEventsRequest) GetMessageId() string {
	if x != nil {
		return x.MessageId
	}
	return ""
}

func (x *ListDeadMessageEventsRequest) GetPage() *psml_pb.PageRequest {
	if x != nil {
		return x.Page
	}
	return nil
}

func (x *ListDeadMessageEventsRequest) GetQuery() *psml_pb.QueryRequest {
	if x != nil {
		return x.Query
	}
	return nil
}

type ListDeadMessageEventsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Events []*dante_pb.DeadMessageEvent `protobuf:"bytes,1,rep,name=events,proto3" json:"events,omitempty"`
	Page   *psml_pb.PageResponse        `protobuf:"bytes,100,opt,name=page,proto3" json:"page,omitempty"`
}

func (x *ListDeadMessageEventsResponse) Reset() {
	*x = ListDeadMessageEventsResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListDeadMessageEventsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListDeadMessageEventsResponse) ProtoMessage() {}

func (x *ListDeadMessageEventsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_o5_dante_v1_service_dead_message_query_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListDeadMessageEventsResponse.ProtoReflect.Descriptor instead.
func (*ListDeadMessageEventsResponse) Descriptor() ([]byte, []int) {
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP(), []int{5}
}

func (x *ListDeadMessageEventsResponse) GetEvents() []*dante_pb.DeadMessageEvent {
	if x != nil {
		return x.Events
	}
	return nil
}

func (x *ListDeadMessageEventsResponse) GetPage() *psml_pb.PageResponse {
	if x != nil {
		return x.Page
	}
	return nil
}

var File_o5_dante_v1_service_dead_message_query_proto protoreflect.FileDescriptor

var file_o5_dante_v1_service_dead_message_query_proto_rawDesc = []byte{
	0x0a, 0x2c, 0x6f, 0x35, 0x2f, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2f, 0x64, 0x65, 0x61, 0x64, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x5f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x13,
	0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x1a, 0x1b, 0x62, 0x75, 0x66, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74,
	0x65, 0x2f, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x61, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x6e, 0x6e,
	0x6f, 0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1c,
	0x6f, 0x35, 0x2f, 0x61, 0x75, 0x74, 0x68, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f, 0x74,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x6f, 0x35,
	0x2f, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x64, 0x65, 0x61, 0x64, 0x5f, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x16, 0x70, 0x73,
	0x6d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76, 0x31, 0x2f, 0x70, 0x61, 0x67, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x17, 0x70, 0x73, 0x6d, 0x2f, 0x6c, 0x69, 0x73, 0x74, 0x2f, 0x76,
	0x31, 0x2f, 0x71, 0x75, 0x65, 0x72, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1e, 0x70,
	0x73, 0x6d, 0x2f, 0x73, 0x74, 0x61, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x61, 0x6e, 0x6e, 0x6f,
	0x74, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x54, 0x0a,
	0x15, 0x47, 0x65, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52,
	0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x2c, 0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48, 0x05, 0x72,
	0x03, 0xb0, 0x01, 0x01, 0x48, 0x00, 0x52, 0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49,
	0x64, 0x88, 0x01, 0x01, 0x42, 0x0d, 0x0a, 0x0b, 0x5f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x5f, 0x69, 0x64, 0x22, 0x90, 0x01, 0x0a, 0x16, 0x47, 0x65, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x3f,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x1d, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65,
	0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x42, 0x06,
	0xba, 0x48, 0x03, 0xc8, 0x01, 0x01, 0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0x35, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1d, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65,
	0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x52, 0x06,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x22, 0x78, 0x0a, 0x17, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65,
	0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x18, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61,
	0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12,
	0x2f, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x65, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x22, 0x84, 0x01, 0x0a, 0x18, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x39, 0x0a,
	0x08, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x1d, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x44, 0x65,
	0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x53, 0x74, 0x61, 0x74, 0x65, 0x52, 0x08,
	0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x12, 0x2d, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65,
	0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73,
	0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x22, 0xa6, 0x01, 0x0a, 0x1c, 0x4c, 0x69, 0x73, 0x74,
	0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74,
	0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x27, 0x0a, 0x0a, 0x6d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x42, 0x08, 0xba, 0x48,
	0x05, 0x72, 0x03, 0xb0, 0x01, 0x01, 0x52, 0x09, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x49,
	0x64, 0x12, 0x2c, 0x0a, 0x04, 0x70, 0x61, 0x67, 0x65, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x18, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61,
	0x67, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x12,
	0x2f, 0x0a, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79, 0x18, 0x65, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19,
	0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69, 0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x51, 0x75, 0x65,
	0x72, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x52, 0x05, 0x71, 0x75, 0x65, 0x72, 0x79,
	0x22, 0x85, 0x01, 0x0a, 0x1d, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x35, 0x0a, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x1d, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31,
	0x2e, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e,
	0x74, 0x52, 0x06, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x2d, 0x0a, 0x04, 0x70, 0x61, 0x67,
	0x65, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x70, 0x73, 0x6d, 0x2e, 0x6c, 0x69,
	0x73, 0x74, 0x2e, 0x76, 0x31, 0x2e, 0x50, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x52, 0x04, 0x70, 0x61, 0x67, 0x65, 0x32, 0xba, 0x04, 0x0a, 0x17, 0x44, 0x65, 0x61,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x51, 0x75, 0x65, 0x72, 0x79, 0x53, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x12, 0xa8, 0x01, 0x0a, 0x0e, 0x47, 0x65, 0x74, 0x44, 0x65, 0x61, 0x64,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x2a, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e,
	0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65,
	0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x1a, 0x2b, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76,
	0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x47, 0x65, 0x74, 0x44, 0x65, 0x61,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x3d, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x22, 0x12, 0x20, 0x2f, 0x64, 0x61, 0x6e, 0x74, 0x65,
	0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2f, 0x7b, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x7d, 0xf2, 0xe8, 0x81, 0xd9, 0x02, 0x0f,
	0x08, 0x01, 0x22, 0x0b, 0x64, 0x65, 0x61, 0x64, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12,
	0xa2, 0x01, 0x0a, 0x10, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73,
	0x61, 0x67, 0x65, 0x73, 0x12, 0x2c, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e,
	0x76, 0x31, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44,
	0x65, 0x61, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x2d, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31,
	0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x31, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x16, 0x12, 0x14, 0x2f, 0x64, 0x61, 0x6e, 0x74,
	0x65, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x73, 0xf2,
	0xe8, 0x81, 0xd9, 0x02, 0x0f, 0x10, 0x01, 0x22, 0x0b, 0x64, 0x65, 0x61, 0x64, 0x6d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x12, 0xc4, 0x01, 0x0a, 0x15, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61,
	0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x12, 0x31,
	0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61, 0x64, 0x4d, 0x65, 0x73,
	0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x32, 0x2e, 0x6f, 0x35, 0x2e, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2e, 0x76, 0x31, 0x2e,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x44, 0x65, 0x61, 0x64,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x45, 0x76, 0x65, 0x6e, 0x74, 0x73, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x44, 0x82, 0xd3, 0xe4, 0x93, 0x02, 0x29, 0x12, 0x27, 0x2f,
	0x64, 0x61, 0x6e, 0x74, 0x65, 0x2f, 0x76, 0x31, 0x2f, 0x71, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61,
	0x67, 0x65, 0x2f, 0x7b, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x5f, 0x69, 0x64, 0x7d, 0x2f,
	0x65, 0x76, 0x65, 0x6e, 0x74, 0x73, 0xf2, 0xe8, 0x81, 0xd9, 0x02, 0x0f, 0x18, 0x01, 0x22, 0x0b,
	0x64, 0x65, 0x61, 0x64, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x1a, 0x08, 0xaa, 0xb7, 0xf5,
	0xe0, 0x01, 0x02, 0x5a, 0x00, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e,
	0x63, 0x6f, 0x6d, 0x2f, 0x70, 0x65, 0x6e, 0x74, 0x6f, 0x70, 0x73, 0x2f, 0x64, 0x61, 0x6e, 0x74,
	0x65, 0x2f, 0x67, 0x65, 0x6e, 0x2f, 0x6f, 0x35, 0x2f, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x2f, 0x76,
	0x31, 0x2f, 0x64, 0x61, 0x6e, 0x74, 0x65, 0x5f, 0x73, 0x70, 0x62, 0x62, 0x06, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x33,
}

var (
	file_o5_dante_v1_service_dead_message_query_proto_rawDescOnce sync.Once
	file_o5_dante_v1_service_dead_message_query_proto_rawDescData = file_o5_dante_v1_service_dead_message_query_proto_rawDesc
)

func file_o5_dante_v1_service_dead_message_query_proto_rawDescGZIP() []byte {
	file_o5_dante_v1_service_dead_message_query_proto_rawDescOnce.Do(func() {
		file_o5_dante_v1_service_dead_message_query_proto_rawDescData = protoimpl.X.CompressGZIP(file_o5_dante_v1_service_dead_message_query_proto_rawDescData)
	})
	return file_o5_dante_v1_service_dead_message_query_proto_rawDescData
}

var file_o5_dante_v1_service_dead_message_query_proto_msgTypes = make([]protoimpl.MessageInfo, 6)
var file_o5_dante_v1_service_dead_message_query_proto_goTypes = []interface{}{
	(*GetDeadMessageRequest)(nil),         // 0: o5.dante.v1.service.GetDeadMessageRequest
	(*GetDeadMessageResponse)(nil),        // 1: o5.dante.v1.service.GetDeadMessageResponse
	(*ListDeadMessagesRequest)(nil),       // 2: o5.dante.v1.service.ListDeadMessagesRequest
	(*ListDeadMessagesResponse)(nil),      // 3: o5.dante.v1.service.ListDeadMessagesResponse
	(*ListDeadMessageEventsRequest)(nil),  // 4: o5.dante.v1.service.ListDeadMessageEventsRequest
	(*ListDeadMessageEventsResponse)(nil), // 5: o5.dante.v1.service.ListDeadMessageEventsResponse
	(*dante_pb.DeadMessageState)(nil),     // 6: o5.dante.v1.DeadMessageState
	(*dante_pb.DeadMessageEvent)(nil),     // 7: o5.dante.v1.DeadMessageEvent
	(*psml_pb.PageRequest)(nil),           // 8: psm.list.v1.PageRequest
	(*psml_pb.QueryRequest)(nil),          // 9: psm.list.v1.QueryRequest
	(*psml_pb.PageResponse)(nil),          // 10: psm.list.v1.PageResponse
}
var file_o5_dante_v1_service_dead_message_query_proto_depIdxs = []int32{
	6,  // 0: o5.dante.v1.service.GetDeadMessageResponse.message:type_name -> o5.dante.v1.DeadMessageState
	7,  // 1: o5.dante.v1.service.GetDeadMessageResponse.events:type_name -> o5.dante.v1.DeadMessageEvent
	8,  // 2: o5.dante.v1.service.ListDeadMessagesRequest.page:type_name -> psm.list.v1.PageRequest
	9,  // 3: o5.dante.v1.service.ListDeadMessagesRequest.query:type_name -> psm.list.v1.QueryRequest
	6,  // 4: o5.dante.v1.service.ListDeadMessagesResponse.messages:type_name -> o5.dante.v1.DeadMessageState
	10, // 5: o5.dante.v1.service.ListDeadMessagesResponse.page:type_name -> psm.list.v1.PageResponse
	8,  // 6: o5.dante.v1.service.ListDeadMessageEventsRequest.page:type_name -> psm.list.v1.PageRequest
	9,  // 7: o5.dante.v1.service.ListDeadMessageEventsRequest.query:type_name -> psm.list.v1.QueryRequest
	7,  // 8: o5.dante.v1.service.ListDeadMessageEventsResponse.events:type_name -> o5.dante.v1.DeadMessageEvent
	10, // 9: o5.dante.v1.service.ListDeadMessageEventsResponse.page:type_name -> psm.list.v1.PageResponse
	0,  // 10: o5.dante.v1.service.DeadMessageQueryService.GetDeadMessage:input_type -> o5.dante.v1.service.GetDeadMessageRequest
	2,  // 11: o5.dante.v1.service.DeadMessageQueryService.ListDeadMessages:input_type -> o5.dante.v1.service.ListDeadMessagesRequest
	4,  // 12: o5.dante.v1.service.DeadMessageQueryService.ListDeadMessageEvents:input_type -> o5.dante.v1.service.ListDeadMessageEventsRequest
	1,  // 13: o5.dante.v1.service.DeadMessageQueryService.GetDeadMessage:output_type -> o5.dante.v1.service.GetDeadMessageResponse
	3,  // 14: o5.dante.v1.service.DeadMessageQueryService.ListDeadMessages:output_type -> o5.dante.v1.service.ListDeadMessagesResponse
	5,  // 15: o5.dante.v1.service.DeadMessageQueryService.ListDeadMessageEvents:output_type -> o5.dante.v1.service.ListDeadMessageEventsResponse
	13, // [13:16] is the sub-list for method output_type
	10, // [10:13] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_o5_dante_v1_service_dead_message_query_proto_init() }
func file_o5_dante_v1_service_dead_message_query_proto_init() {
	if File_o5_dante_v1_service_dead_message_query_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetDeadMessageRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GetDeadMessageResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDeadMessagesRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDeadMessagesResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDeadMessageEventsRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_o5_dante_v1_service_dead_message_query_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListDeadMessageEventsResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_o5_dante_v1_service_dead_message_query_proto_msgTypes[0].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_o5_dante_v1_service_dead_message_query_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   6,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_o5_dante_v1_service_dead_message_query_proto_goTypes,
		DependencyIndexes: file_o5_dante_v1_service_dead_message_query_proto_depIdxs,
		MessageInfos:      file_o5_dante_v1_service_dead_message_query_proto_msgTypes,
	}.Build()
	File_o5_dante_v1_service_dead_message_query_proto = out.File
	file_o5_dante_v1_service_dead_message_query_proto_rawDesc = nil
	file_o5_dante_v1_service_dead_message_query_proto_goTypes = nil
	file_o5_dante_v1_service_dead_message_query_proto_depIdxs = nil
}
