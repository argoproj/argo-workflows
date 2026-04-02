package apiclient

import (
	"context"

	"google.golang.org/grpc"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type argoKubeCronWorkflowServiceClient struct {
	delegate cronworkflowpkg.CronWorkflowServiceServer
}

var _ cronworkflowpkg.CronWorkflowServiceClient = &argoKubeCronWorkflowServiceClient{}

func (c *argoKubeCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.LintCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.CreateCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	return c.delegate.ListCronWorkflows(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.GetCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.UpdateCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	return c.delegate.DeleteCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) ResumeCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.ResumeCronWorkflow(ctx, req)
}

func (c *argoKubeCronWorkflowServiceClient) SuspendCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return c.delegate.SuspendCronWorkflow(ctx, req)
}
