package http1

import (
	"context"

	"google.golang.org/grpc"

	workflowtemplatepkg "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflowtemplate"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type WorkflowTemplateServiceClient = Facade

func (h WorkflowTemplateServiceClient) CreateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h WorkflowTemplateServiceClient) GetWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Get(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) ListWorkflowTemplates(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplateList, error) {
	out := &wfv1.WorkflowTemplateList{}
	return out, h.Get(in, out, "/api/v1/workflow-templates/{namespace}")
}

func (h WorkflowTemplateServiceClient) UpdateWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Put(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) DeleteWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	out := &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}
	return out, h.Delete(in, out, "/api/v1/workflow-templates/{namespace}/{name}")
}

func (h WorkflowTemplateServiceClient) LintWorkflowTemplate(_ context.Context, in *workflowtemplatepkg.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*wfv1.WorkflowTemplate, error) {
	out := &wfv1.WorkflowTemplate{}
	return out, h.Post(in, out, "/api/v1/workflow-templates/{namespace}/lint")
}
