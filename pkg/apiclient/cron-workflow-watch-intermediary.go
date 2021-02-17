package apiclient

import (
	"context"
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

type cronWorkflowWatchIntermediary struct {
	abstractIntermediary
	events chan *cronworkflow.CronWorkflowWatchEvent
}

func (w cronWorkflowWatchIntermediary) Send(e *cronworkflow.CronWorkflowWatchEvent) error {
	w.events <- e
	return nil
}

func (w cronWorkflowWatchIntermediary) Recv() (*cronworkflow.CronWorkflowWatchEvent, error) {
	select {
	case e := <-w.error:
		return nil, e
	case event := <-w.events:
		return event, nil
	}
}

func newCronWorkflowWatchIntermediary(ctx context.Context) *cronWorkflowWatchIntermediary {
	return &cronWorkflowWatchIntermediary{newAbstractIntermediary(ctx), make(chan *cronworkflow.CronWorkflowWatchEvent)}
}
