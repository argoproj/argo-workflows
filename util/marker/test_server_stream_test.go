package marker

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testServerStream struct {
}

// the only purpose of this line is to make sure testServerStream implements ServerStream
var _ grpc.ServerStream = &testServerStream{}

func (t testServerStream) SetHeader(md metadata.MD) error {
	panic("implement me")
}

func (t testServerStream) SendHeader(md metadata.MD) error {
	panic("implement me")
}

func (t testServerStream) SetTrailer(md metadata.MD) {
	panic("implement me")
}

func (t testServerStream) Context() context.Context {
	return context.Background()
}

func (t testServerStream) SendMsg(m interface{}) error {
	panic("implement me")
}

func (t testServerStream) RecvMsg(m interface{}) error {
	panic("implement me")
}
