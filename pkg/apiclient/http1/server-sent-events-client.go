package http1

import (
	"bufio"
	"context"
	"encoding/json"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// serverSentEventsClient provides a RecvEvent func to make getting Server-Sent Events (SSE)
// simple and consistent
type serverSentEventsClient struct {
	//nolint: containedctx
	ctx    context.Context
	reader *bufio.Reader
}

func (c serverSentEventsClient) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c serverSentEventsClient) Trailer() metadata.MD {
	panic("implement me")
}

func (c serverSentEventsClient) CloseSend() error {
	panic("implement me")
}

func (c serverSentEventsClient) Context() context.Context {
	return c.ctx
}

func (c serverSentEventsClient) SendMsg(any) error {
	panic("implement me")
}

func (c serverSentEventsClient) RecvMsg(any) error {
	panic("implement me")
}

const prefixLength = len("data: ")

func (c serverSentEventsClient) RecvEvent(v any) error {
	log := logging.RequireLoggerFromContext(c.ctx)
	for {
		line, err := c.reader.ReadBytes('\n')
		if err != nil {
			return err
		}
		log.Debug(c.ctx, string(line))
		// each line must be prefixed with `data: `, if not we just ignore it
		// maybe empty line for example
		if len(line) <= prefixLength {
			continue
		}
		// the actual data itself always has a `{"result": v}` field
		x := struct {
			Result any `json:"result"`
		}{
			Result: v,
		}
		return json.Unmarshal(line[prefixLength:], &x)
	}
}

var _ grpc.ClientStream = &serverSentEventsClient{}
