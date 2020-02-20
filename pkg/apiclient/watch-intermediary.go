package apiclient

import (
	"context"
	"io"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

// The "Poison pill pattern" to tell the channel to close.
var closeTheChan *workflowpkg.WorkflowWatchEvent

type watchIntermediary struct {
	events chan *workflowpkg.WorkflowWatchEvent
}

func newWatchIntermediary() *watchIntermediary {
	return &watchIntermediary{make(chan *workflowpkg.WorkflowWatchEvent, 512)}
}

func (c watchIntermediary) Send(e *workflowpkg.WorkflowWatchEvent) error {
	c.events <- e
	return nil
}

func (c watchIntermediary) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	e := <-c.events
	if e == closeTheChan {
		return nil, io.EOF
	}
	return e, nil
}

func (c watchIntermediary) Header() (metadata.MD, error) {
	panic("implement me")
}

func (c watchIntermediary) Trailer() metadata.MD {
	panic("implement me")
}

func (c watchIntermediary) CloseSend() error {
	c.events <- closeTheChan
	return nil
}

func (c watchIntermediary) Context() context.Context {
	panic("implement me")
}

func (c watchIntermediary) SendMsg(m interface{}) error {
	panic("implement me")
}

func (c watchIntermediary) RecvMsg(m interface{}) error {
	panic("implement me")
}
