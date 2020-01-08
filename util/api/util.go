package api

import (
	"context"

	apiwf "github.com/argoproj/argo/cmd/server/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func SubmitWorkflowToAPIServer(apiGRPCClient apiwf.WorkflowServiceClient, ctx context.Context, wf *wfv1.Workflow, dryRun bool) (*wfv1.Workflow, error) {
	return apiGRPCClient.CreateWorkflow(ctx, &apiwf.WorkflowCreateRequest{Workflow: wf, ServerDryRun: dryRun})
}
