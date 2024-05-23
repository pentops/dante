package dynamictype

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type ProtoJSON struct {
	protojson.MarshalOptions
	protojson.UnmarshalOptions
}

type Resolver interface {
	protoregistry.ExtensionTypeResolver
	protoregistry.MessageTypeResolver
}

func NewProtoJSON(resolver Resolver) *ProtoJSON {
	return &ProtoJSON{
		MarshalOptions: protojson.MarshalOptions{
			Resolver: resolver,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			Resolver: resolver,
		},
	}
}

func (pj ProtoJSON) Marshal(msg proto.Message) ([]byte, error) {
	return pj.MarshalOptions.Marshal(msg)
}

func (pj ProtoJSON) Unmarshal(b []byte, msg proto.Message) error {
	return pj.UnmarshalOptions.Unmarshal(b, msg)
}

type TypeRegistry struct {
	messages map[string]protoreflect.MessageDescriptor

	// leave nil, will panic if used.
	protoregistry.ExtensionTypeResolver
}

func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		messages: make(map[string]protoreflect.MessageDescriptor),
	}
}

func (r *TypeRegistry) FindMessageByName(field protoreflect.FullName) (protoreflect.MessageType, error) {
	var descriptor protoreflect.MessageDescriptor
	descriptor, ok := r.messages[string(field)]

	if !ok {
		td, err := protoregistry.GlobalTypes.FindMessageByName(field)
		if err != nil {
			return nil, fmt.Errorf("couldn't find message by name: %s", field)
		}
		descriptor = td.Descriptor()
	}

	msg := dynamicpb.NewMessageType(descriptor)

	return msg, nil
}

func (r *TypeRegistry) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	message := protoreflect.FullName(url)
	if i := strings.LastIndexByte(url, '/'); i >= 0 {
		message = message[i+len("/"):]
	}
	return r.FindMessageByName(message)
}

func (r *TypeRegistry) AddMessage(msg protoreflect.MessageDescriptor) {
	r.messages[string(msg.FullName())] = msg
	subMessages := msg.Messages()
	for idx := 0; idx < subMessages.Len(); idx++ {
		r.AddMessage(subMessages.Get(idx))
	}
}

func (r *TypeRegistry) AddFileDescriptor(fds *descriptorpb.FileDescriptorSet) error {
	fp, err := protodesc.NewFiles(fds)
	if err != nil {
		return fmt.Errorf("Couldn't convert to registry file: %v", err.Error())
	}

	fp.RangeFiles(func(a protoreflect.FileDescriptor) bool {
		fileMessages := a.Messages()
		for idx := 0; idx < fileMessages.Len(); idx++ {
			msg := fileMessages.Get(idx)
			r.AddMessage(msg)
		}

		return true
	})
	return nil
}

func (r *TypeRegistry) LoadProtoFromFile(fileName string) error {
	protoFile, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("couldn't read file: %v", err.Error())
	}

	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(protoFile, fds)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal file protofile: %v", err.Error())
	}

	return r.AddFileDescriptor(fds)
}

func (r *TypeRegistry) LoadExternalProtobufs(src string) error {
	if len(src) == 0 {
		return nil
	}

	res, err := http.Get(src)
	if err != nil {
		return fmt.Errorf("couldn't get file from %v: %v", src, err.Error())
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		res.Body.Close()
		return fmt.Errorf("couldn't read body: %v", err.Error())
	}
	res.Body.Close()

	if res.StatusCode > 299 {
		return fmt.Errorf("got status code %v instead of 2xx", res.StatusCode)
	}

	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(body, fds)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal file protofile: %v", err.Error())
	}

	return r.AddFileDescriptor(fds)
}
