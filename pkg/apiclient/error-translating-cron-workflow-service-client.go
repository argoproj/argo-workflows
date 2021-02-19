package apiclient

import (
	"context"

	"google.golang.org/grpc"

	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
)

type errorTranslatingCronWorkflowServiceClient struct {
	delegate cronworkflowpkg.CronWorkflowServiceClient
}

var _ cronworkflowpkg.CronWorkflowServiceClient = &errorTranslatingCronWorkflowServiceClient{}

func (c *errorTranslatingCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.LintCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.CreateCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	workflows, err := c.delegate.ListCronWorkflows(ctx, req)
	return workflows, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.GetCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.UpdateCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	workflow, err := c.delegate.DeleteCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) ResumeCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.ResumeCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}

func (c *errorTranslatingCronWorkflowServiceClient) SuspendCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.SuspendCronWorkflow(ctx, req)
	return workflow, grpcutil.TranslateError(err)
}
