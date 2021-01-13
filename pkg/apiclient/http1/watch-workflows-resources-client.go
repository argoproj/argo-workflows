package http1

import (
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type watchWorkflowsResourcesClient struct{ serverSentEventsClient }

func (f watchWorkflowsResourcesClient) Recv() (*workflowpkg.WorkflowResourceWatchEvent, error) {
	v := &workflowpkg.WorkflowResourceWatchEvent{}
	return v, f.RecvEvent(v)
}
