package apiclient

import (
	"context"

	"google.golang.org/grpc/metadata"
	v1 "k8s.io/api/core/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
)

type workflowWatchIntermediary struct {
	abstractIntermediary
	events chan *workflowpkg.WorkflowWatchEvent
}

func (w workflowWatchIntermediary) Send(e *workflowpkg.WorkflowWatchEvent) error {
	w.events <- e
	return nil
}

func (w workflowWatchIntermediary) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	select {
	case e := <-w.error:
		return nil, e
	case event := <-w.events:
		return event, nil
	}
}

func (w *workflowWatchIntermediary) SendHeader(metadata.MD) error {
	// We invoke `SendHeader` in order to eagerly flush headers to allow us to send period
	// keepalives when using HTTP/1 and Server Sent Events, so we need to implement this here,
	// though we don't use the meta for anything.
	return nil
}

func newWorkflowWatchIntermediary(ctx context.Context) *workflowWatchIntermediary {
	return &workflowWatchIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.WorkflowWatchEvent)}
}

type eventWatchIntermediary struct {
	abstractIntermediary
	events chan *v1.Event
}

func (w eventWatchIntermediary) Send(e *v1.Event) error {
	w.events <- e
	return nil
}

func (w eventWatchIntermediary) Recv() (*v1.Event, error) {
	select {
	case e := <-w.error:
		return nil, e
	case event := <-w.events:
		return event, nil
	}
}

func (w *eventWatchIntermediary) SendHeader(metadata.MD) error {
	// We invoke `SendHeader` in order to eagerly flush headers to allow us to send period
	// keepalives when using HTTP/1 and Server Sent Events, so we need to implement this here,
	// though we don't use the meta for anything.
	return nil
}

func newEventWatchIntermediary(ctx context.Context) *eventWatchIntermediary {
	return &eventWatchIntermediary{newAbstractIntermediary(ctx), make(chan *v1.Event)}
}
