package apiclient

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	grpcutil "github.com/argoproj/argo/util/grpc"
)

type errorTranslatingWorkflowTemplateServiceClient struct {
	delegate workflowtemplatepkg.WorkflowTemplateServiceClient
}

var _ workflowtemplatepkg.WorkflowTemplateServiceClient = &errorTranslatingWorkflowTemplateServiceClient{}

func (a *errorTranslatingWorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.CreateWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a *errorTranslatingWorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.GetWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a *errorTranslatingWorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	templates, err := a.delegate.ListWorkflowTemplates(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return templates, nil
}

func (a *errorTranslatingWorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.UpdateWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a *errorTranslatingWorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	template, err := a.delegate.DeleteWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}

func (a *errorTranslatingWorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	template, err := a.delegate.LintWorkflowTemplate(ctx, req)
	if err != nil {
		return nil, grpcutil.TranslateError(err)
	}
	return template, nil
}
