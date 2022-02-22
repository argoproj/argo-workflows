package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
)

type errorTranslatingWorkflowTemplateServiceClient struct {
	delegate workflowtemplatepkg.WorkflowTemplateServiceClient
}

var _ workflowtemplatepkg.WorkflowTemplateServiceClient = &errorTranslatingWorkflowTemplateServiceClient{}

func (a *errorTranslatingWorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.CreateWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a *errorTranslatingWorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.GetWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a *errorTranslatingWorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	templates, err := a.delegate.ListWorkflowTemplates(ctx, req)
	return templates, grpcutil.TranslateError(err)
}

func (a *errorTranslatingWorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.UpdateWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a *errorTranslatingWorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	template, err := a.delegate.DeleteWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}

func (a *errorTranslatingWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.LintWorkflowTemplate(ctx, req)
	return template, grpcutil.TranslateError(err)
}
