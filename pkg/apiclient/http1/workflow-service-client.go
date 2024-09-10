package http1

import (
	"context"

	"google.golang.org/grpc"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type WorkflowServiceClient = Facade

func (h WorkflowServiceClient) CreateWorkflow(ctx context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(ctx, in, out, "/api/v1/workflows/{namespace}")
}

func (h WorkflowServiceClient) GetWorkflow(ctx context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(ctx, in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h WorkflowServiceClient) ListWorkflows(ctx context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(ctx, in, out, "/api/v1/workflows/{namespace}")
}

func (h WorkflowServiceClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	reader, err := h.EventStreamReader(ctx, in, "/api/v1/workflow-events/{namespace}")
	if err != nil {
		return nil, err
	}
	return watchWorkflowsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) WatchEvents(ctx context.Context, in *workflowpkg.WatchEventsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	reader, err := h.EventStreamReader(ctx, in, "/api/v1/stream/events/{namespace}")
	if err != nil {
		return nil, err
	}
	return eventWatchClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) DeleteWorkflow(ctx context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	out := &workflowpkg.WorkflowDeleteResponse{}
	return out, h.Delete(ctx, in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h WorkflowServiceClient) RetryWorkflow(ctx context.Context, in *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/retry")
}

func (h WorkflowServiceClient) ResubmitWorkflow(ctx context.Context, in *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/resubmit")
}

func (h WorkflowServiceClient) ResumeWorkflow(ctx context.Context, in *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/resume")
}

func (h WorkflowServiceClient) SuspendWorkflow(ctx context.Context, in *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/suspend")
}

func (h WorkflowServiceClient) TerminateWorkflow(ctx context.Context, in *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/terminate")
}

func (h WorkflowServiceClient) StopWorkflow(ctx context.Context, in *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/stop")
}

func (h WorkflowServiceClient) SetWorkflow(ctx context.Context, in *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/set")
}

func (h WorkflowServiceClient) ApproveWorkflow(ctx context.Context, in *workflowpkg.WorkflowApproveRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(ctx, in, out, "/api/v1/workflows/{namespace}/{name}/approve")
}

func (h WorkflowServiceClient) LintWorkflow(ctx context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(ctx, in, out, "/api/v1/workflows/{namespace}/lint")
}

func (h WorkflowServiceClient) PodLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	reader, err := h.EventStreamReader(ctx, in, "/api/v1/workflows/{namespace}/{name}/{podName}/log")
	if err != nil {
		return nil, err
	}
	return &podLogsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) WorkflowLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WorkflowLogsClient, error) {
	reader, err := h.EventStreamReader(ctx, in, "/api/v1/workflows/{namespace}/{name}/log")
	if err != nil {
		return nil, err
	}
	return &podLogsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) SubmitWorkflow(ctx context.Context, in *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(ctx, in, out, "/api/v1/workflows/{namespace}/submit")
}
