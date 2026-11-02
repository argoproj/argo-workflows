package workflow

import (
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/jsonpb" //nolint:staticcheck // grpc-gateway v1 JSONPBMarshaler requires this package
	corev1 "k8s.io/api/core/v1"
)

// eventProtoAdapter wraps corev1.Event to satisfy the proto.Message interface.
// k8s v0.35+ removed ProtoMessage() from core types, but grpc-gateway v1
// generated code requires all streamed response types to implement proto.Message.
// It also implements jsonpb.JSONPBMarshaler so the grpc-gateway jsonpb marshaler
// serializes the underlying Event as standard JSON rather than using proto reflection.
type eventProtoAdapter struct {
	*corev1.Event
}

func (e *eventProtoAdapter) ProtoMessage()  {}
func (e *eventProtoAdapter) Reset()         { *e.Event = corev1.Event{} }
func (e *eventProtoAdapter) String() string { return fmt.Sprintf("%v", e.Event) }
func (e *eventProtoAdapter) MarshalJSONPB(*jsonpb.Marshaler) ([]byte, error) {
	return json.Marshal(e.Event)
}

func wrapEventAsProtoMessage(event *corev1.Event, err error) (*eventProtoAdapter, error) {
	if err != nil {
		return nil, err
	}
	return &eventProtoAdapter{Event: event}, nil
}
