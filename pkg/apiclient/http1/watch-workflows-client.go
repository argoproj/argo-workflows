package http1

import (
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

type watchWorkflowsClient struct{ serverSentEventsClient }

func (f watchWorkflowsClient) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	v := &workflowpkg.WorkflowWatchEvent{}
	return v, f.RecvEvent(v)
}
