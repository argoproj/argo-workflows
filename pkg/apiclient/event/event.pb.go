// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: pkg/apiclient/event/event.proto

package event

import (
	context "context"
	fmt "fmt"
	v1alpha1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	proto "github.com/gogo/protobuf/proto"
	_ "google.golang.org/genproto/googleapis/api/annotations"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	io "io"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

type EventRequest struct {
	// The namespace for the event. This can be empty if the client has cluster scoped permissions.
	// If empty, then the event is "broadcast" to workflow event binding in all namespaces.
	Namespace string `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	// Optional discriminator for the event. This should almost always be empty.
	// Used for edge-cases where the event payload alone is not provide enough information to discriminate the event.
	// This MUST NOT be used as security mechanism, e.g. to allow two clients to use the same access token, or
	// to support webhooks on unsecured server. Instead, use access tokens.
	// This is made available as `discriminator` in the event binding selector (`/spec/event/selector)`
	Discriminator string `protobuf:"bytes,2,opt,name=discriminator,proto3" json:"discriminator,omitempty"`
	// The event itself can be any data.
	Payload              *v1alpha1.Item `protobuf:"bytes,3,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *EventRequest) Reset()         { *m = EventRequest{} }
func (m *EventRequest) String() string { return proto.CompactTextString(m) }
func (*EventRequest) ProtoMessage()    {}
func (*EventRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_d80a0d2509a47d1c, []int{0}
}
func (m *EventRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventRequest.Merge(m, src)
}
func (m *EventRequest) XXX_Size() int {
	return m.Size()
}
func (m *EventRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_EventRequest.DiscardUnknown(m)
}

var xxx_messageInfo_EventRequest proto.InternalMessageInfo

func (m *EventRequest) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *EventRequest) GetDiscriminator() string {
	if m != nil {
		return m.Discriminator
	}
	return ""
}

func (m *EventRequest) GetPayload() *v1alpha1.Item {
	if m != nil {
		return m.Payload
	}
	return nil
}

type EventResponse struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *EventResponse) Reset()         { *m = EventResponse{} }
func (m *EventResponse) String() string { return proto.CompactTextString(m) }
func (*EventResponse) ProtoMessage()    {}
func (*EventResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_d80a0d2509a47d1c, []int{1}
}
func (m *EventResponse) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventResponse.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventResponse.Merge(m, src)
}
func (m *EventResponse) XXX_Size() int {
	return m.Size()
}
func (m *EventResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_EventResponse.DiscardUnknown(m)
}

var xxx_messageInfo_EventResponse proto.InternalMessageInfo

type ListWorkflowEventBindingsRequest struct {
	Namespace            string          `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	ListOptions          *v1.ListOptions `protobuf:"bytes,2,opt,name=listOptions,proto3" json:"listOptions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *ListWorkflowEventBindingsRequest) Reset()         { *m = ListWorkflowEventBindingsRequest{} }
func (m *ListWorkflowEventBindingsRequest) String() string { return proto.CompactTextString(m) }
func (*ListWorkflowEventBindingsRequest) ProtoMessage()    {}
func (*ListWorkflowEventBindingsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_d80a0d2509a47d1c, []int{2}
}
func (m *ListWorkflowEventBindingsRequest) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *ListWorkflowEventBindingsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_ListWorkflowEventBindingsRequest.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *ListWorkflowEventBindingsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ListWorkflowEventBindingsRequest.Merge(m, src)
}
func (m *ListWorkflowEventBindingsRequest) XXX_Size() int {
	return m.Size()
}
func (m *ListWorkflowEventBindingsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ListWorkflowEventBindingsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ListWorkflowEventBindingsRequest proto.InternalMessageInfo

func (m *ListWorkflowEventBindingsRequest) GetNamespace() string {
	if m != nil {
		return m.Namespace
	}
	return ""
}

func (m *ListWorkflowEventBindingsRequest) GetListOptions() *v1.ListOptions {
	if m != nil {
		return m.ListOptions
	}
	return nil
}

func init() {
	proto.RegisterType((*EventRequest)(nil), "event.EventRequest")
	proto.RegisterType((*EventResponse)(nil), "event.EventResponse")
	proto.RegisterType((*ListWorkflowEventBindingsRequest)(nil), "event.ListWorkflowEventBindingsRequest")
}

func init() { proto.RegisterFile("pkg/apiclient/event/event.proto", fileDescriptor_d80a0d2509a47d1c) }

var fileDescriptor_d80a0d2509a47d1c = []byte{
	// 476 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x93, 0xd1, 0x8a, 0x13, 0x31,
	0x14, 0x86, 0x49, 0x45, 0x65, 0xd3, 0x5d, 0x84, 0xe8, 0x45, 0x2d, 0x4b, 0x2d, 0x83, 0xe0, 0xa2,
	0x36, 0x61, 0x5a, 0x05, 0x59, 0xbd, 0x5a, 0xf1, 0x42, 0x58, 0x50, 0x66, 0x41, 0xc1, 0xbb, 0x74,
	0x7a, 0x9c, 0xc6, 0xce, 0x24, 0x31, 0xc9, 0x66, 0x59, 0x96, 0xbd, 0xf1, 0x15, 0xc4, 0x97, 0xf0,
	0x49, 0x04, 0x6f, 0x04, 0x7d, 0x00, 0x29, 0x3e, 0x88, 0x4c, 0x3a, 0xb3, 0x33, 0x45, 0x65, 0x65,
	0x6f, 0x4a, 0x7a, 0x72, 0xce, 0xc9, 0xf7, 0xff, 0xe7, 0x0c, 0xbe, 0xa5, 0x17, 0x19, 0xe3, 0x5a,
	0xa4, 0xb9, 0x00, 0xe9, 0x18, 0xf8, 0xb3, 0x5f, 0xaa, 0x8d, 0x72, 0x8a, 0x5c, 0x0e, 0x7f, 0xfa,
	0xdb, 0x99, 0x52, 0x59, 0x0e, 0x65, 0x2a, 0xe3, 0x52, 0x2a, 0xc7, 0x9d, 0x50, 0xd2, 0xae, 0x92,
	0xfa, 0x0f, 0x16, 0x8f, 0x2c, 0x15, 0xaa, 0xbc, 0x2d, 0x78, 0x3a, 0x17, 0x12, 0xcc, 0x31, 0xab,
	0x3a, 0x5b, 0x56, 0x80, 0xe3, 0xcc, 0xc7, 0x2c, 0x03, 0x09, 0x86, 0x3b, 0x98, 0x55, 0x55, 0x4f,
	0x33, 0xe1, 0xe6, 0x87, 0x53, 0x9a, 0xaa, 0x82, 0x71, 0x93, 0x29, 0x6d, 0xd4, 0xbb, 0x70, 0x68,
	0x4a, 0x8f, 0x94, 0x59, 0xbc, 0xcd, 0xd5, 0x11, 0xf3, 0x31, 0xcf, 0xf5, 0x9c, 0xff, 0xd1, 0x24,
	0xfa, 0x8c, 0xf0, 0xe6, 0xb3, 0x12, 0x31, 0x81, 0xf7, 0x87, 0x60, 0x1d, 0xd9, 0xc6, 0x1b, 0x92,
	0x17, 0x60, 0x35, 0x4f, 0xa1, 0x87, 0x86, 0x68, 0x67, 0x23, 0x69, 0x02, 0xe4, 0x36, 0xde, 0x9a,
	0x09, 0x9b, 0x1a, 0x51, 0x08, 0xc9, 0x9d, 0x32, 0xbd, 0x4e, 0xc8, 0x58, 0x0f, 0x92, 0x57, 0xf8,
	0xaa, 0xe6, 0xc7, 0xb9, 0xe2, 0xb3, 0xde, 0xa5, 0x21, 0xda, 0xe9, 0x8e, 0x9f, 0xd0, 0x86, 0x95,
	0xd6, 0xac, 0xe1, 0x40, 0xfd, 0x84, 0xea, 0x45, 0x46, 0x4b, 0x5c, 0x5a, 0xe3, 0xd2, 0x1a, 0x97,
	0x3e, 0x77, 0x50, 0x24, 0x75, 0xb3, 0xe8, 0x1a, 0xde, 0xaa, 0x58, 0xad, 0x56, 0xd2, 0x42, 0xf4,
	0x09, 0xe1, 0xe1, 0xbe, 0xb0, 0xee, 0x75, 0x55, 0x18, 0x6e, 0xf7, 0x84, 0x9c, 0x09, 0x99, 0xd9,
	0xff, 0x53, 0x74, 0x80, 0xbb, 0xb9, 0xb0, 0xee, 0x85, 0x0e, 0x03, 0x09, 0x7a, 0xba, 0xe3, 0x98,
	0xae, 0x26, 0x42, 0xdb, 0x13, 0x69, 0x38, 0xcb, 0x89, 0x50, 0x1f, 0xd3, 0xfd, 0xa6, 0x30, 0x69,
	0x77, 0x19, 0xff, 0xe8, 0x54, 0xae, 0x1e, 0x80, 0xf1, 0x22, 0x05, 0xe2, 0xf1, 0x66, 0x02, 0x29,
	0x08, 0x0f, 0x21, 0x4c, 0xae, 0xd3, 0xd5, 0x92, 0xb4, 0xad, 0xef, 0xdf, 0x58, 0x0f, 0x56, 0x1a,
	0x1f, 0x7f, 0xf8, 0xfe, 0xeb, 0x63, 0xe7, 0x61, 0x74, 0x37, 0x2c, 0x8f, 0x8f, 0x57, 0xeb, 0x65,
	0xd9, 0xc9, 0x99, 0x86, 0x53, 0x76, 0xb2, 0xe6, 0xff, 0xe9, 0x6e, 0xed, 0x18, 0xf9, 0x8a, 0xf0,
	0xcd, 0x7f, 0x1a, 0x44, 0xee, 0x54, 0x0f, 0x9e, 0x67, 0x61, 0xff, 0xe5, 0x45, 0xe7, 0xf7, 0xb7,
	0xae, 0xe5, 0x6b, 0xd1, 0x24, 0xa8, 0x1a, 0x91, 0x7b, 0xb5, 0xaa, 0xba, 0x76, 0x14, 0x90, 0x46,
	0xd3, 0x8a, 0xa0, 0x2d, 0x73, 0x6f, 0xf7, 0xcb, 0x72, 0x80, 0xbe, 0x2d, 0x07, 0xe8, 0xe7, 0x72,
	0x80, 0xde, 0xdc, 0x3f, 0x6f, 0xff, 0xdb, 0x1f, 0xe5, 0xf4, 0x4a, 0xd8, 0xf7, 0xc9, 0xef, 0x00,
	0x00, 0x00, 0xff, 0xff, 0xc2, 0xeb, 0x67, 0xf6, 0xb2, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// EventServiceClient is the client API for EventService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type EventServiceClient interface {
	ReceiveEvent(ctx context.Context, in *EventRequest, opts ...grpc.CallOption) (*EventResponse, error)
	ListWorkflowEventBindings(ctx context.Context, in *ListWorkflowEventBindingsRequest, opts ...grpc.CallOption) (*v1alpha1.WorkflowEventBindingList, error)
}

type eventServiceClient struct {
	cc *grpc.ClientConn
}

func NewEventServiceClient(cc *grpc.ClientConn) EventServiceClient {
	return &eventServiceClient{cc}
}

func (c *eventServiceClient) ReceiveEvent(ctx context.Context, in *EventRequest, opts ...grpc.CallOption) (*EventResponse, error) {
	out := new(EventResponse)
	err := c.cc.Invoke(ctx, "/event.EventService/ReceiveEvent", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *eventServiceClient) ListWorkflowEventBindings(ctx context.Context, in *ListWorkflowEventBindingsRequest, opts ...grpc.CallOption) (*v1alpha1.WorkflowEventBindingList, error) {
	out := new(v1alpha1.WorkflowEventBindingList)
	err := c.cc.Invoke(ctx, "/event.EventService/ListWorkflowEventBindings", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EventServiceServer is the server API for EventService service.
type EventServiceServer interface {
	ReceiveEvent(context.Context, *EventRequest) (*EventResponse, error)
	ListWorkflowEventBindings(context.Context, *ListWorkflowEventBindingsRequest) (*v1alpha1.WorkflowEventBindingList, error)
}

// UnimplementedEventServiceServer can be embedded to have forward compatible implementations.
type UnimplementedEventServiceServer struct {
}

func (*UnimplementedEventServiceServer) ReceiveEvent(ctx context.Context, req *EventRequest) (*EventResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReceiveEvent not implemented")
}
func (*UnimplementedEventServiceServer) ListWorkflowEventBindings(ctx context.Context, req *ListWorkflowEventBindingsRequest) (*v1alpha1.WorkflowEventBindingList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListWorkflowEventBindings not implemented")
}

func RegisterEventServiceServer(s *grpc.Server, srv EventServiceServer) {
	s.RegisterService(&_EventService_serviceDesc, srv)
}

func _EventService_ReceiveEvent_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EventRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).ReceiveEvent(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/ReceiveEvent",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).ReceiveEvent(ctx, req.(*EventRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _EventService_ListWorkflowEventBindings_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ListWorkflowEventBindingsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(EventServiceServer).ListWorkflowEventBindings(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/event.EventService/ListWorkflowEventBindings",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(EventServiceServer).ListWorkflowEventBindings(ctx, req.(*ListWorkflowEventBindingsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _EventService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "event.EventService",
	HandlerType: (*EventServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ReceiveEvent",
			Handler:    _EventService_ReceiveEvent_Handler,
		},
		{
			MethodName: "ListWorkflowEventBindings",
			Handler:    _EventService_ListWorkflowEventBindings_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/apiclient/event/event.proto",
}

func (m *EventRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.Payload != nil {
		{
			size, err := m.Payload.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEvent(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Discriminator) > 0 {
		i -= len(m.Discriminator)
		copy(dAtA[i:], m.Discriminator)
		i = encodeVarintEvent(dAtA, i, uint64(len(m.Discriminator)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Namespace) > 0 {
		i -= len(m.Namespace)
		copy(dAtA[i:], m.Namespace)
		i = encodeVarintEvent(dAtA, i, uint64(len(m.Namespace)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *EventResponse) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventResponse) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventResponse) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	return len(dAtA) - i, nil
}

func (m *ListWorkflowEventBindingsRequest) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *ListWorkflowEventBindingsRequest) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *ListWorkflowEventBindingsRequest) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		i -= len(m.XXX_unrecognized)
		copy(dAtA[i:], m.XXX_unrecognized)
	}
	if m.ListOptions != nil {
		{
			size, err := m.ListOptions.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEvent(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if len(m.Namespace) > 0 {
		i -= len(m.Namespace)
		copy(dAtA[i:], m.Namespace)
		i = encodeVarintEvent(dAtA, i, uint64(len(m.Namespace)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintEvent(dAtA []byte, offset int, v uint64) int {
	offset -= sovEvent(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *EventRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Namespace)
	if l > 0 {
		n += 1 + l + sovEvent(uint64(l))
	}
	l = len(m.Discriminator)
	if l > 0 {
		n += 1 + l + sovEvent(uint64(l))
	}
	if m.Payload != nil {
		l = m.Payload.Size()
		n += 1 + l + sovEvent(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *EventResponse) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func (m *ListWorkflowEventBindingsRequest) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Namespace)
	if l > 0 {
		n += 1 + l + sovEvent(uint64(l))
	}
	if m.ListOptions != nil {
		l = m.ListOptions.Size()
		n += 1 + l + sovEvent(uint64(l))
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovEvent(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEvent(x uint64) (n int) {
	return sovEvent(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *EventRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvent
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Namespace", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Namespace = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Discriminator", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Discriminator = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Payload", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Payload == nil {
				m.Payload = &v1alpha1.Item{}
			}
			if err := m.Payload.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEvent(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *EventResponse) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvent
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventResponse: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventResponse: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		default:
			iNdEx = preIndex
			skippy, err := skipEvent(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *ListWorkflowEventBindingsRequest) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvent
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: ListWorkflowEventBindingsRequest: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: ListWorkflowEventBindingsRequest: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Namespace", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Namespace = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ListOptions", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEvent
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEvent
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.ListOptions == nil {
				m.ListOptions = &v1.ListOptions{}
			}
			if err := m.ListOptions.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEvent(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if skippy < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) < 0 {
				return ErrInvalidLengthEvent
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, dAtA[iNdEx:iNdEx+skippy]...)
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipEvent(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEvent
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEvent
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthEvent
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEvent
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEvent
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEvent        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEvent          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEvent = fmt.Errorf("proto: unexpected end of group")
)
