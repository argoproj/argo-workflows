package http

import (
	"bufio"
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type clientStream struct {
	ctx context.Context
	reader *bufio.Reader
}

func (c clientStream) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c clientStream) Trailer() metadata.MD {
	panic("implement me")
}

func (c clientStream) CloseSend() error {
	panic("implement me")
}

func (c clientStream) Context() context.Context {
	return c.ctx
}

func (c clientStream) SendMsg(interface{}) error {
	panic("implement me")
}

func (c clientStream) RecvMsg(interface{}) error {
	panic("implement me")
}

var _ grpc.ClientStream = &clientStream{}
