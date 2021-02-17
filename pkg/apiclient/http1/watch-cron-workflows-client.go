package http1

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
)

type watchCronWorkflowsClient struct{ serverSentEventsClient }

func (f watchCronWorkflowsClient) Recv() (*cronworkflow.CronWorkflowWatchEvent, error) {
	v := &cronworkflow.CronWorkflowWatchEvent{}
	return v, f.RecvEvent(v)
}
