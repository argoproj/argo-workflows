package http1

import (
	"context"

	"google.golang.org/grpc"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type WorkflowServiceClient = Facade

func (h WorkflowServiceClient) CreateWorkflow(_ context.Context, in *workflowpkg.WorkflowCreateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}")
}

func (h WorkflowServiceClient) GetWorkflow(_ context.Context, in *workflowpkg.WorkflowGetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Get(in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h WorkflowServiceClient) ListWorkflows(_ context.Context, in *workflowpkg.WorkflowListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	out := &wfv1.WorkflowList{}
	return out, h.Get(in, out, "/api/v1/workflows/{namespace}")
}

func (h WorkflowServiceClient) WatchWorkflows(ctx context.Context, in *workflowpkg.WatchWorkflowsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/workflow-events/{namespace}")
	if err != nil {
		return nil, err
	}
	return watchWorkflowsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) WatchEvents(ctx context.Context, in *workflowpkg.WatchEventsRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/stream/events/{namespace}")
	if err != nil {
		return nil, err
	}
	return eventWatchClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) DeleteWorkflow(_ context.Context, in *workflowpkg.WorkflowDeleteRequest, _ ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	out := &workflowpkg.WorkflowDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/workflows/{namespace}/{name}")
}

func (h WorkflowServiceClient) RetryWorkflow(_ context.Context, in *workflowpkg.WorkflowRetryRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/retry")
}

func (h WorkflowServiceClient) ResubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowResubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resubmit")
}

func (h WorkflowServiceClient) ResumeWorkflow(_ context.Context, in *workflowpkg.WorkflowResumeRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/resume")
}

func (h WorkflowServiceClient) SuspendWorkflow(_ context.Context, in *workflowpkg.WorkflowSuspendRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/suspend")
}

func (h WorkflowServiceClient) TerminateWorkflow(_ context.Context, in *workflowpkg.WorkflowTerminateRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/terminate")
}

func (h WorkflowServiceClient) StopWorkflow(_ context.Context, in *workflowpkg.WorkflowStopRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/stop")
}

func (h WorkflowServiceClient) SetWorkflow(_ context.Context, in *workflowpkg.WorkflowSetRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Put(in, out, "/api/v1/workflows/{namespace}/{name}/set")
}

func (h WorkflowServiceClient) LintWorkflow(_ context.Context, in *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/lint")
}

func (h WorkflowServiceClient) PodLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/workflows/{namespace}/{name}/{podName}/log")
	if err != nil {
		return nil, err
	}
	return &podLogsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) WorkflowLogs(ctx context.Context, in *workflowpkg.WorkflowLogRequest, _ ...grpc.CallOption) (workflowpkg.WorkflowService_WorkflowLogsClient, error) {
	reader, err := h.EventStreamReader(in, "/api/v1/workflows/{namespace}/{name}/log")
	if err != nil {
		return nil, err
	}
	return &podLogsClient{serverSentEventsClient{ctx, reader}}, nil
}

func (h WorkflowServiceClient) SubmitWorkflow(_ context.Context, in *workflowpkg.WorkflowSubmitRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	out := &wfv1.Workflow{}
	return out, h.Post(in, out, "/api/v1/workflows/{namespace}/submit")
}
