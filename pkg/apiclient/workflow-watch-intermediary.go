package apiclient

import (
	"context"
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

func newWorkflowWatchIntermediary(ctx context.Context) *workflowWatchIntermediary {
	return &workflowWatchIntermediary{newAbstractIntermediary(ctx), make(chan *workflowpkg.WorkflowWatchEvent)}
}
