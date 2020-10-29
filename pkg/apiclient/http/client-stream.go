package http

import (
	"bufio"
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type clientStream struct {
	ctx    context.Context
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

const prefixLength = len("data: ")

func (f clientStream) RecvEvent(v interface{}) error {
	for {
		data, err := f.reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		if len(data) <= prefixLength {
			continue
		}
		return json.Unmarshal(data[prefixLength:], v)
	}
}

var _ grpc.ClientStream = &clientStream{}
