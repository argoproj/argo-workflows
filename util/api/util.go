package api

import (
	"context"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	apiwf "github.com/argoproj/argo/server/workflow"
)

func SubmitWorkflowToAPIServer(apiGRPCClient apiwf.WorkflowServiceClient, ctx context.Context, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error) {

	wfReq := apiwf.WorkflowCreateRequest{
		Namespace:    wf.Namespace,
		Workflow:     wf,
		ServerDryRun: dryRun,
	}
	return apiGRPCClient.CreateWorkflow(ctx, &wfReq)
}
