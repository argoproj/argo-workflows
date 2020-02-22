package apiclient

import (
	"context"
	"io"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

// The "Poison pill pattern" to tell the channel to close.
var closeTheWorkflowWatchEventChan *workflowpkg.WorkflowWatchEvent

type watchIntermediary struct {
	context context.Context
	events  chan *workflowpkg.WorkflowWatchEvent
}

func (c watchIntermediary) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c watchIntermediary) Trailer() metadata.MD {
	panic("implement me")
}

func (c watchIntermediary) SetHeader(metadata.MD) error {
	panic("implement me")
}

func (c watchIntermediary) SendHeader(metadata.MD) error {
	panic("implement me")
}

func (c watchIntermediary) SetTrailer(metadata.MD) {
	panic("implement me")
}

func (c watchIntermediary) Context() context.Context {
	return c.context
}

func (c watchIntermediary) SendMsg(m interface{}) error {
	panic("implement me")
}

func (c watchIntermediary) RecvMsg(m interface{}) error {
	panic("implement me")
}

func newWatchIntermediary(ctx context.Context) *watchIntermediary {
	return &watchIntermediary{ctx, make(chan *workflowpkg.WorkflowWatchEvent, 512)}
}

func (c watchIntermediary) Send(e *workflowpkg.WorkflowWatchEvent) error {
	c.events <- e
	return nil
}

func (c watchIntermediary) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	e := <-c.events
	if e == closeTheWorkflowWatchEventChan {
		return nil, io.EOF
	}
	return e, nil
}

func (c watchIntermediary) CloseSend() error {
	c.events <- closeTheWorkflowWatchEventChan
	return nil
}
