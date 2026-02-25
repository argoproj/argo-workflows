package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v4/workflow/validate"
)

type OfflineWorkflowTemplateServiceClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap offlineWorkflowTemplateGetterMap
}

var _ workflowtemplatepkg.WorkflowTemplateServiceClient = &OfflineWorkflowTemplateServiceClient{}

func (o OfflineWorkflowTemplateServiceClient) CreateWorkflowTemplate(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineWorkflowTemplateServiceClient) GetWorkflowTemplate(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineWorkflowTemplateServiceClient) ListWorkflowTemplates(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	return nil, ErrOffline
}

func (o OfflineWorkflowTemplateServiceClient) UpdateWorkflowTemplate(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, ErrOffline
}

func (o OfflineWorkflowTemplateServiceClient) DeleteWorkflowTemplate(_ context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	return nil, ErrOffline
}

func (o OfflineWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	err := validate.ValidateWorkflowTemplate(ctx, o.namespacedWorkflowTemplateGetterMap.GetNamespaceGetter(req.Namespace), o.clusterWorkflowTemplateGetter, req.Template, nil, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return req.Template, nil
}
