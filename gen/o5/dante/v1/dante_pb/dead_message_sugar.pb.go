// Code generated by protoc-gen-go-sugar. DO NOT EDIT.

package dante_pb

import (
	driver "database/sql/driver"
	fmt "fmt"
)

// DeadMessageEventType is a oneof wrapper
type DeadMessageEventTypeKey string

const (
	DeadMessageEvent_Notified DeadMessageEventTypeKey = "notified"
	DeadMessageEvent_Updated  DeadMessageEventTypeKey = "updated"
	DeadMessageEvent_Replayed DeadMessageEventTypeKey = "replayed"
	DeadMessageEvent_Rejected DeadMessageEventTypeKey = "rejected"
)

func (x *DeadMessageEventType) TypeKey() (DeadMessageEventTypeKey, bool) {
	switch x.Type.(type) {
	case *DeadMessageEventType_Notified_:
		return DeadMessageEvent_Notified, true
	case *DeadMessageEventType_Updated_:
		return DeadMessageEvent_Updated, true
	case *DeadMessageEventType_Replayed_:
		return DeadMessageEvent_Replayed, true
	case *DeadMessageEventType_Rejected_:
		return DeadMessageEvent_Rejected, true
	default:
		return "", false
	}
}

type IsDeadMessageEventTypeWrappedType interface {
	TypeKey() DeadMessageEventTypeKey
}

func (x *DeadMessageEventType) Set(val IsDeadMessageEventTypeWrappedType) {
	switch v := val.(type) {
	case *DeadMessageEventType_Notified:
		x.Type = &DeadMessageEventType_Notified_{Notified: v}
	case *DeadMessageEventType_Updated:
		x.Type = &DeadMessageEventType_Updated_{Updated: v}
	case *DeadMessageEventType_Replayed:
		x.Type = &DeadMessageEventType_Replayed_{Replayed: v}
	case *DeadMessageEventType_Rejected:
		x.Type = &DeadMessageEventType_Rejected_{Rejected: v}
	}
}
func (x *DeadMessageEventType) Get() IsDeadMessageEventTypeWrappedType {
	switch v := x.Type.(type) {
	case *DeadMessageEventType_Notified_:
		return v.Notified
	case *DeadMessageEventType_Updated_:
		return v.Updated
	case *DeadMessageEventType_Replayed_:
		return v.Replayed
	case *DeadMessageEventType_Rejected_:
		return v.Rejected
	default:
		return nil
	}
}
func (x *DeadMessageEventType_Notified) TypeKey() DeadMessageEventTypeKey {
	return DeadMessageEvent_Notified
}
func (x *DeadMessageEventType_Updated) TypeKey() DeadMessageEventTypeKey {
	return DeadMessageEvent_Updated
}
func (x *DeadMessageEventType_Replayed) TypeKey() DeadMessageEventTypeKey {
	return DeadMessageEvent_Replayed
}
func (x *DeadMessageEventType_Rejected) TypeKey() DeadMessageEventTypeKey {
	return DeadMessageEvent_Rejected
}

type IsDeadMessageEventType_Type = isDeadMessageEventType_Type

// MessageStatus
const (
	MessageStatus_UNSPECIFIED MessageStatus = 0
	MessageStatus_CREATED     MessageStatus = 1
	MessageStatus_UPDATED     MessageStatus = 2
	MessageStatus_REPLAYED    MessageStatus = 3
	MessageStatus_REJECTED    MessageStatus = 4
)

var (
	MessageStatus_name_short = map[int32]string{
		0: "UNSPECIFIED",
		1: "CREATED",
		2: "UPDATED",
		3: "REPLAYED",
		4: "REJECTED",
	}
	MessageStatus_value_short = map[string]int32{
		"UNSPECIFIED": 0,
		"CREATED":     1,
		"UPDATED":     2,
		"REPLAYED":    3,
		"REJECTED":    4,
	}
	MessageStatus_value_either = map[string]int32{
		"UNSPECIFIED":                0,
		"MESSAGE_STATUS_UNSPECIFIED": 0,
		"CREATED":                    1,
		"MESSAGE_STATUS_CREATED":     1,
		"UPDATED":                    2,
		"MESSAGE_STATUS_UPDATED":     2,
		"REPLAYED":                   3,
		"MESSAGE_STATUS_REPLAYED":    3,
		"REJECTED":                   4,
		"MESSAGE_STATUS_REJECTED":    4,
	}
)

// ShortString returns the un-prefixed string representation of the enum value
func (x MessageStatus) ShortString() string {
	return MessageStatus_name_short[int32(x)]
}
func (x MessageStatus) Value() (driver.Value, error) {
	return []uint8(x.ShortString()), nil
}
func (x *MessageStatus) Scan(value interface{}) error {
	var strVal string
	switch vt := value.(type) {
	case []uint8:
		strVal = string(vt)
	case string:
		strVal = vt
	default:
		return fmt.Errorf("invalid type %T", value)
	}
	val := MessageStatus_value_either[strVal]
	*x = MessageStatus(val)
	return nil
}

// Urgency
const (
	Urgency_UNSPECIFIED Urgency = 0
	Urgency_LOW         Urgency = 1
	Urgency_MEDIUM      Urgency = 2
	Urgency_HIGH        Urgency = 3
)

var (
	Urgency_name_short = map[int32]string{
		0: "UNSPECIFIED",
		1: "LOW",
		2: "MEDIUM",
		3: "HIGH",
	}
	Urgency_value_short = map[string]int32{
		"UNSPECIFIED": 0,
		"LOW":         1,
		"MEDIUM":      2,
		"HIGH":        3,
	}
	Urgency_value_either = map[string]int32{
		"UNSPECIFIED":         0,
		"URGENCY_UNSPECIFIED": 0,
		"LOW":                 1,
		"URGENCY_LOW":         1,
		"MEDIUM":              2,
		"URGENCY_MEDIUM":      2,
		"HIGH":                3,
		"URGENCY_HIGH":        3,
	}
)

// ShortString returns the un-prefixed string representation of the enum value
func (x Urgency) ShortString() string {
	return Urgency_name_short[int32(x)]
}
func (x Urgency) Value() (driver.Value, error) {
	return []uint8(x.ShortString()), nil
}
func (x *Urgency) Scan(value interface{}) error {
	var strVal string
	switch vt := value.(type) {
	case []uint8:
		strVal = string(vt)
	case string:
		strVal = vt
	default:
		return fmt.Errorf("invalid type %T", value)
	}
	val := Urgency_value_either[strVal]
	*x = Urgency(val)
	return nil
}
