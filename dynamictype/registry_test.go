package dynamictype

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestDynamicLoad(t *testing.T) {
	msgBase64 := "CgNmb28="
	// a message with a single field, number 1, value is 'foo'

	msgBytes, err := base64.StdEncoding.DecodeString(msgBase64)
	if err != nil {
		t.Fatal(err)
	}

	msg := &anypb.Any{
		Value:   msgBytes,
		TypeUrl: "type.googleapis.com/namespace.v1.Foo",
	}

	ps := &descriptorpb.FileDescriptorSet{}
	outFile := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("namespace/v1/foo.proto"),
		Package: proto.String("namespace.v1"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Foo"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("field"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				JsonName: proto.String("field"),
			}},
		}},
	}
	ps.File = append(ps.File, outFile)

	types := NewTypeRegistry()
	if err := types.AddFileDescriptor(ps); err != nil {
		t.Fatal(err.Error())
	}

	msg_json, err := protojson.MarshalOptions{
		Resolver: types,
	}.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("json: %s", string(msg_json))

	proto := map[string]interface{}{}
	if err := json.Unmarshal(msg_json, &proto); err != nil {
		t.Fatal(err)
	}
	type_url, ok := proto["@type"].(string)
	if !ok {
		t.Fatal("type_url not found")
	}
	if type_url != "type.googleapis.com/namespace.v1.Foo" {
		t.Fatalf("type_url is not correct: %s", type_url)
	}
	value, ok := proto["field"].(string)
	if !ok {
		t.Fatal("field not found")
	}
	if value != "foo" {
		t.Fatalf("field is not correct: %s", value)
	}

}
