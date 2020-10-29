package http

import (
	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
)

type watchWorkflowsClient struct{ clientStream }

func (f watchWorkflowsClient) Recv() (*workflowpkg.WorkflowWatchEvent, error) {
	v := &workflowpkg.WorkflowWatchEvent{}
	return v, f.RecvEvent(v)
}
