package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type argoKubeWorkflowTemplateServiceClient struct {
	delegate workflowtemplatepkg.WorkflowTemplateServiceServer
}

var _ workflowtemplatepkg.WorkflowTemplateServiceClient = &argoKubeWorkflowTemplateServiceClient{}

func (a *argoKubeWorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return a.delegate.CreateWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return a.delegate.GetWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	return a.delegate.ListWorkflowTemplates(ctx, req)
}

func (a *argoKubeWorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return a.delegate.UpdateWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	return a.delegate.DeleteWorkflowTemplate(ctx, req)
}

func (a *argoKubeWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	return a.delegate.LintWorkflowTemplate(ctx, req)
}
