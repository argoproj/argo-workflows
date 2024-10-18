package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/templateresolution"
	"github.com/argoproj/argo-workflows/v3/workflow/validate"
)

type OfflineWorkflowTemplateServiceClient struct {
	clusterWorkflowTemplateGetter       templateresolution.ClusterWorkflowTemplateGetter
	namespacedWorkflowTemplateGetterMap offlineWorkflowTemplateGetterMap
}

var _ workflowtemplatepkg.WorkflowTemplateServiceClient = &OfflineWorkflowTemplateServiceClient{}

func (o OfflineWorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	return nil, OfflineErr
}

func (o OfflineWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	err := validate.ValidateWorkflowTemplate(o.namespacedWorkflowTemplateGetterMap.GetNamespaceGetter(req.Namespace), o.clusterWorkflowTemplateGetter, req.Template, nil, validate.ValidateOpts{Lint: true})
	if err != nil {
		return nil, err
	}
	return req.Template, nil
}
