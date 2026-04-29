package apiclient

import (
	"context"
	"io"

	"google.golang.org/grpc"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type argoKubeWorkflowServiceClient struct {
	delegate workflowpkg.WorkflowServiceServer
}

var _ workflowpkg.WorkflowServiceClient = &argoKubeWorkflowServiceClient{}

func (c *argoKubeWorkflowServiceClient) CreateWorkflow(ctx context.Context, req *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.CreateWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) GetWorkflow(ctx context.Context, req *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.GetWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) ListWorkflows(ctx context.Context, req *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowList, error) {
	return c.delegate.ListWorkflows(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) WatchWorkflows(ctx context.Context, req *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	intermediary := newWorkflowWatchIntermediary(ctx)
	go func() {
		defer intermediary.cancel()
		err := c.delegate.WatchWorkflows(req, intermediary)
		if err != nil {
			intermediary.error <- err
		} else {
			intermediary.error <- io.EOF
		}
	}()
	return intermediary, nil
}

func (c *argoKubeWorkflowServiceClient) WatchEvents(ctx context.Context, req *workflowpkg.WatchEventsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	intermediary := newEventWatchIntermediary(ctx)
	go func() {
		defer intermediary.cancel()
		err := c.delegate.WatchEvents(req, intermediary)
		if err != nil {
			intermediary.error <- err
		} else {
			intermediary.error <- io.EOF
		}
	}()
	return intermediary, nil
}

func (c *argoKubeWorkflowServiceClient) DeleteWorkflow(ctx context.Context, req *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	return c.delegate.DeleteWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) RetryWorkflow(ctx context.Context, req *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.RetryWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) ResubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.ResubmitWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) ResumeWorkflow(ctx context.Context, req *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.ResumeWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) SuspendWorkflow(ctx context.Context, req *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.SuspendWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) StopWorkflow(ctx context.Context, req *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.StopWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) SetWorkflow(ctx context.Context, req *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.SetWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) TerminateWorkflow(ctx context.Context, req *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.TerminateWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) LintWorkflow(ctx context.Context, req *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.LintWorkflow(ctx, req)
}

func (c *argoKubeWorkflowServiceClient) logs(ctx context.Context, req *workflowpkg.WorkflowLogRequest, f func(*workflowpkg.WorkflowLogRequest, *logsIntermediary) error) (workflowpkg.WorkflowService_PodLogsClient, error) {
	intermediary := newLogsIntermediary(ctx)
	go func() {
		defer intermediary.cancel()
		err := f(req, intermediary)
		if err != nil {
			intermediary.error <- err
		} else {
			intermediary.error <- io.EOF
		}
	}()
	return intermediary, nil
}

func (c *argoKubeWorkflowServiceClient) PodLogs(ctx context.Context, req *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	return c.logs(ctx, req, func(req *workflowpkg.WorkflowLogRequest, i *logsIntermediary) error {
		return c.delegate.PodLogs(req, i)
	})
}

func (c *argoKubeWorkflowServiceClient) WorkflowLogs(ctx context.Context, req *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WorkflowLogsClient, error) {
	return c.logs(ctx, req, func(req *workflowpkg.WorkflowLogRequest, i *logsIntermediary) error {
		return c.delegate.WorkflowLogs(req, i)
	})
}

func (c *argoKubeWorkflowServiceClient) SubmitWorkflow(ctx context.Context, req *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*v1alpha1.Workflow, error) {
	return c.delegate.SubmitWorkflow(ctx, req)
}
