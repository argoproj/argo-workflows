package apiclient

import (
	"context"

	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
)

type classicWorkflowTemplateServiceClient struct {
	versioned.Interface
}

func (c classicWorkflowTemplateServiceClient) CreateWorkflowTemplate(_ context.Context, _ *workflowtemplate.WorkflowTemplateCreateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	panic("implement me")
}

func (c classicWorkflowTemplateServiceClient) GetWorkflowTemplate(_ context.Context, req *workflowtemplate.WorkflowTemplateGetRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	options := metav1.GetOptions{}
	if req.GetOptions != nil {
		options = *req.GetOptions
	}
	return c.ArgoprojV1alpha1().WorkflowTemplates(req.GetNamespace()).Get(req.GetName(), options)
}

func (c classicWorkflowTemplateServiceClient) ListWorkflowTemplates(_ context.Context, _ *workflowtemplate.WorkflowTemplateListRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplateList, error) {
	panic("implement me")
}

func (c classicWorkflowTemplateServiceClient) UpdateWorkflowTemplate(_ context.Context, _ *workflowtemplate.WorkflowTemplateUpdateRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	panic("implement me")
}

func (c classicWorkflowTemplateServiceClient) DeleteWorkflowTemplate(_ context.Context, _ *workflowtemplate.WorkflowTemplateDeleteRequest, _ ...grpc.CallOption) (*workflowtemplate.WorkflowTemplateDeleteResponse, error) {
	panic("implement me")
}

func (c classicWorkflowTemplateServiceClient) LintWorkflowTemplate(_ context.Context, _ *workflowtemplate.WorkflowTemplateLintRequest, _ ...grpc.CallOption) (*v1alpha1.WorkflowTemplate, error) {
	panic("implement me")
}
