package http1

import (
	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
)

type watchWorkflowPodClient struct{ serverSentEventsClient }

func (f watchWorkflowPodClient) Recv() (*workflowpkg.WorkflowPodWatchEvent, error) {
	v := &workflowpkg.WorkflowPodWatchEvent{}
	return v, f.RecvEvent(v)
}
