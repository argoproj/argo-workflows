package apiclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	workflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

var OfflineErr = fmt.Errorf("not supported when you are in offline mode")

type OfflineWorkflowServiceClient struct{}

var _ workflowpkg.WorkflowServiceClient = &OfflineWorkflowServiceClient{}

func (o OfflineWorkflowServiceClient) CreateWorkflow(context.Context, *workflowpkg.WorkflowCreateRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) GetWorkflow(context.Context, *workflowpkg.WorkflowGetRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) ListWorkflows(context.Context, *workflowpkg.WorkflowListRequest, ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) WatchWorkflows(context.Context, *workflowpkg.WatchWorkflowsRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) WatchEvents(context.Context, *workflowpkg.WatchEventsRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_WatchEventsClient, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) DeleteWorkflow(context.Context, *workflowpkg.WorkflowDeleteRequest, ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) RetryWorkflow(context.Context, *workflowpkg.WorkflowRetryRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) ResubmitWorkflow(context.Context, *workflowpkg.WorkflowResubmitRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) ResumeWorkflow(context.Context, *workflowpkg.WorkflowResumeRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) SuspendWorkflow(context.Context, *workflowpkg.WorkflowSuspendRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) TerminateWorkflow(context.Context, *workflowpkg.WorkflowTerminateRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) StopWorkflow(context.Context, *workflowpkg.WorkflowStopRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) SetWorkflow(context.Context, *workflowpkg.WorkflowSetRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

type offlineWorkflowTemplateNamespacedGetter struct{}

func (w offlineWorkflowTemplateNamespacedGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	return nil, OfflineErr
}

type offlineClusterWorkflowTemplateNamespacedGetter struct{}

func (o offlineClusterWorkflowTemplateNamespacedGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) LintWorkflow(_ context.Context, req *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	_, err := validate.ValidateWorkflow(&offlineWorkflowTemplateNamespacedGetter{}, &offlineClusterWorkflowTemplateNamespacedGetter{}, req.Workflow, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return req.Workflow, nil
}

func (o OfflineWorkflowServiceClient) PodLogs(context.Context, *workflowpkg.WorkflowLogRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) WorkflowLogs(context.Context, *workflowpkg.WorkflowLogRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_WorkflowLogsClient, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowServiceClient) SubmitWorkflow(context.Context, *workflowpkg.WorkflowSubmitRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}
