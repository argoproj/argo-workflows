package workflowtemplate

import (
	"context"
	"fmt"
	"github.com/argoproj/argo/server/auth"
	"sort"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type WorkflowTemplateServer struct {
}

func NewWorkflowTemplateServer() WorkflowTemplateServiceServer {
	return &WorkflowTemplateServer{}
}

func (wts *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, req *WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, fmt.Errorf("workflow template was not found in the request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	return wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Create(req.Template)

}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, req *WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Get(req.Name, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return wfTmpl, err
}

func (wts *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, req *WorkflowTemplateListRequest) (*v1alpha1.WorkflowTemplateList, error) {
	wfClient := auth.GetWfClient(ctx)
	options := v1.ListOptions{}
	if req.ListOptions != nil {
		options = *req.ListOptions
	}
	wfList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).List(options)
	if err != nil {
		return nil, err
	}

	sort.Sort(wfList.Items)

	return wfList, nil
}

func (wts *WorkflowTemplateServer) DeleteWorkflowTemplate(ctx context.Context, req *WorkflowTemplateDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)

	err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Delete(req.Name, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &WorkflowDeleteResponse{}, nil
}

func (wts *WorkflowTemplateServer) LintWorkflowTemplate(ctx context.Context, req *WorkflowTemplateLintRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	return req.Template, nil
}

func (wts *WorkflowTemplateServer) UpdateWorkflowTemplate(ctx context.Context, req *WorkflowTemplateUpdateRequest) (*v1alpha1.WorkflowTemplate, error) {
	if req.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	res, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Update(req.Template)
	return res, err
}
