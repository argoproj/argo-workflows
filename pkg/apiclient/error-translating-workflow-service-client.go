package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo/util/grpc"
)

type errorTranslatingWorkflowServiceClient struct {
	delegate workflowpkg.WorkflowServiceClient
}

var _ workflowpkg.WorkflowServiceClient = &errorTranslatingWorkflowServiceClient{}

func (c *errorTranslatingWorkflowServiceClient) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.CreateWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) GetWorkflow(ctx context.Context, req *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.GetWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowList, error) {
	workflows, err := c.delegate.ListWorkflows(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflows, nil
}

func (c *errorTranslatingWorkflowServiceClient) WatchWorkflows(ctx context.Context, req *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	workflows, err := c.delegate.WatchWorkflows(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflows, nil
}

func (c *errorTranslatingWorkflowServiceClient) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	workflow, err := c.delegate.DeleteWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.RetryWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.ResubmitWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.ResumeWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.SuspendWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.StopWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.TerminateWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) LintWorkflow(ctx context.Context, req *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.LintWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}

func (c *errorTranslatingWorkflowServiceClient) PodLogs(ctx context.Context, req *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	logs, err := c.delegate.PodLogs(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return logs, nil
}

func (c *errorTranslatingWorkflowServiceClient) SubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowSubmitRequest, opts ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	workflow, err := c.delegate.SubmitWorkflow(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return workflow, nil
}
