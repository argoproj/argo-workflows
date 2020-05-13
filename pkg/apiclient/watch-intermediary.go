package apiclient

import (
	"context"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
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

func newWatchIntermediary(ctx context.Context) *watchIntermediary {
	return &watchIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.WorkflowWatchEvent)}
}
