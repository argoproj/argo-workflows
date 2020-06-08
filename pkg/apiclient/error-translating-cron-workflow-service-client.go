package apiclient

import (
	"context"

	"google.golang.org/grpc"

	cronworkflowpkg "github.com/argoproj/argo/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo/util/grpc"
)

type errorTranslatingCronWorkflowServiceClient struct {
	delegate cronworkflowpkg.CronWorkflowServiceClient
}

var _ cronworkflowpkg.CronWorkflowServiceClient = &errorTranslatingCronWorkflowServiceClient{}

func (c *errorTranslatingCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.LintCronWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.CreateCronWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	workflows, err := c.delegate.ListCronWorkflows(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflows, nil
}

func (c *errorTranslatingCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.GetCronWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, err
}

func (c *errorTranslatingCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	workflow, err := c.delegate.UpdateCronWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	workflow, err := c.delegate.DeleteCronWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, err
}
