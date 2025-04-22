package apiclient

import (
	"context"

	"google.golang.org/grpc"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

type OfflineCronWorkflowServiceClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap offlineWorkflowTemplateGetterMap
}

var _ cronworkflow.CronWorkflowServiceClient = &OfflineCronWorkflowServiceClient{}

func (o OfflineCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflow.LintCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	err := validate.ValidateCronWorkflow(ctx, o.namespacedWorkflowTemplateGetterMap.GetNamespaceGetter(req.Namespace), o.clusterWorkflowTemplateGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return req.CronWorkflow, nil
}

func (o OfflineCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflow.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflow.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflow.GetCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflow.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflow.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflow.CronWorkflowDeletedResponse, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) ResumeCronWorkflow(ctx context.Context, req *cronworkflow.CronWorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) SuspendCronWorkflow(ctx context.Context, req *cronworkflow.CronWorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}
