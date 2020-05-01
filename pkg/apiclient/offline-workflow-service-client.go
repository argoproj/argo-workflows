package apiclient

import (
	"context"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/validate"
)

type offlineWorkflowServiceClient struct{}

func (o offlineWorkflowServiceClient) CreateWorkflow(context.Context, *workflowpkg.WorkflowCreateRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) GetWorkflow(context.Context, *workflowpkg.WorkflowGetRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) ListWorkflows(context.Context, *workflowpkg.WorkflowListRequest, ...grpc.CallOption) (*wfv1.WorkflowList, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) WatchWorkflows(context.Context, *workflowpkg.WatchWorkflowsRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_WatchWorkflowsClient, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) DeleteWorkflow(context.Context, *workflowpkg.WorkflowDeleteRequest, ...grpc.CallOption) (*workflowpkg.WorkflowDeleteResponse, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) RetryWorkflow(context.Context, *workflowpkg.WorkflowRetryRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) ResubmitWorkflow(context.Context, *workflowpkg.WorkflowResubmitRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) ResumeWorkflow(context.Context, *workflowpkg.WorkflowResumeRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) SuspendWorkflow(context.Context, *workflowpkg.WorkflowSuspendRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) TerminateWorkflow(context.Context, *workflowpkg.WorkflowTerminateRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) StopWorkflow(context.Context, *workflowpkg.WorkflowStopRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}

type offlineWorkflowTemplateNamespacedGetter struct{}

func (w offlineWorkflowTemplateNamespacedGetter) Get(name string) (*wfv1.WorkflowTemplate, error) {
	return &wfv1.WorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
}

type offlineClusterWorkflowTemplateNamespacedGetter struct{}

func (o offlineClusterWorkflowTemplateNamespacedGetter) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	return &wfv1.ClusterWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: name}}, nil
}

func (o offlineWorkflowServiceClient) LintWorkflow(_ context.Context, req *workflowpkg.WorkflowLintRequest, _ ...grpc.CallOption) (*wfv1.Workflow, error) {
	_, err := validate.ValidateWorkflow(&offlineWorkflowTemplateNamespacedGetter{}, &offlineClusterWorkflowTemplateNamespacedGetter{}, req.Workflow, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return req.Workflow, nil
}

func (o offlineWorkflowServiceClient) PodLogs(context.Context, *workflowpkg.WorkflowLogRequest, ...grpc.CallOption) (workflowpkg.WorkflowService_PodLogsClient, error) {
	return nil, OfflineErr
}

func (o offlineWorkflowServiceClient) SubmitWorkflow(context.Context, *workflowpkg.WorkflowSubmitRequest, ...grpc.CallOption) (*wfv1.Workflow, error) {
	return nil, OfflineErr
}
