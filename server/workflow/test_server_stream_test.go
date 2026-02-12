package workflow

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type testServerStream struct {
	//nolint:containedctx
	ctx context.Context
}

var _ grpc.ServerStream = &testServerStream{}

func (t testServerStream) SetHeader(md metadata.MD) error {
	panic("implement me")
}

func (t testServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (t testServerStream) SetTrailer(md metadata.MD) {
	panic("implement me")
}

func (t testServerStream) Context() context.Context {
	return t.ctx
}

func (t testServerStream) SendMsg(any) error {
	panic("implement me")
}

func (t testServerStream) RecvMsg(any) error {
	panic("implement me")
}
