package apiclient

import (
	"context"

	"google.golang.org/grpc/metadata"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
)

type watchIntermediary struct {
	abstractIntermediary
	events chan *workflowpkg.WorkflowWatchEvent
}

func (w watchIntermediary) Send(e *workflowpkg.WorkflowWatchEvent) error {
	w.events <- e
	return nil
}

func (w watchIntermediary) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	select {
	case e := <-w.error:
		return nil, e
	case event := <-w.events:
		return event, nil
	}
}

func (w *watchIntermediary) SendHeader(metadata.MD) error {
	// We invoke `SendHeader` in order to eagerly flush headers to allow us to send period
	// keepalives when using HTTP/1 and Server Sent Events, so we need to implement this here,
	// though we don't use the meta for anything.
	return nil
}

func newWatchIntermediary(ctx context.Context) *watchIntermediary {
	return &watchIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.WorkflowWatchEvent)}
}
