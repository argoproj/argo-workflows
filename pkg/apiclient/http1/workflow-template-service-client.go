package http1

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

type WorkflowTemplateServiceClient = Facade

func (h WorkflowTemplateServiceClient) CreateWorkflowTemplate(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(ctx, in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h WorkflowTemplateServiceClient) GetWorkflowTemplate(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Get(ctx, in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) ListWorkflowTemplates(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	out := &wfv1.WorkflowTemplateList{}
	return out, h.Get(ctx, in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h WorkflowTemplateServiceClient) UpdateWorkflowTemplate(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(ctx, in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) DeleteWorkflowTemplate(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	out := &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}
	return out, h.Delete(ctx, in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) LintWorkflowTemplate(ctx context.Context, in *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(ctx, in, out, "/api/v1/workflow-templates/{namespace}/lint")
}
