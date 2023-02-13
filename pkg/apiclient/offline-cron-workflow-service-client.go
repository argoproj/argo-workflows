package apiclient

import (
	"context"

	"google.golang.org/grpc"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	cronworkflowpkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/cronworkflow"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

type OfflineCronWorkflowServiceClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap map[string]templateresolution.WorkflowTemplateNamespacedGetter
}

var _ cronworkflow.CronWorkflowServiceClient = &OfflineCronWorkflowServiceClient{}

func (o OfflineCronWorkflowServiceClient) LintCronWorkflow(ctx context.Context, req *cronworkflowpkg.LintCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	err := validate.ValidateCronWorkflow(o.namespacedWorkflowTemplateGetterMap[req.Namespace], o.clusterWorkflowTemplateGetter, req.CronWorkflow)
	if err != nil {
		return nil, err
	}
	return req.CronWorkflow, nil
}

func (o OfflineCronWorkflowServiceClient) CreateCronWorkflow(ctx context.Context, req *cronworkflowpkg.CreateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) ListCronWorkflows(ctx context.Context, req *cronworkflowpkg.ListCronWorkflowsRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflowList, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) GetCronWorkflow(ctx context.Context, req *cronworkflowpkg.GetCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) UpdateCronWorkflow(ctx context.Context, req *cronworkflowpkg.UpdateCronWorkflowRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) DeleteCronWorkflow(ctx context.Context, req *cronworkflowpkg.DeleteCronWorkflowRequest, _ ...grpc.CallOption) (*cronworkflowpkg.CronWorkflowDeletedResponse, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) ResumeCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowResumeRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}

func (o OfflineCronWorkflowServiceClient) SuspendCronWorkflow(ctx context.Context, req *cronworkflowpkg.CronWorkflowSuspendRequest, _ ...grpc.CallOption) (*v1alpha1.CronWorkflow, error) {
	return nil, OfflineErr
}
