package api

import (
	"context"

	"github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func SubmitWorkflowToAPIServer(apiGRPCClient workflow.WorkflowServiceClient, ctx context.Context, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error) {

	wfReq := workflow.WorkflowCreateRequest{
		Namespace:    wf.Namespace,
		Workflow:     wf,
		ServerDryRun: dryRun,
	}
	return apiGRPCClient.CreateWorkflow(ctx, &wfReq)
}
