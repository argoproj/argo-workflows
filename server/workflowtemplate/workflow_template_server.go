package workflowtemplate

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowtemplatepkg "github.com/argoproj/argo/pkg/apiclient/workflowtemplate"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/server/auth"
	"github.com/argoproj/argo/util/resource"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type WorkflowTemplateServer struct {
}

func NewWorkflowTemplateServer() workflowtemplatepkg.WorkflowTemplateServiceServer {
	return &WorkflowTemplateServer{}
}

func (wts *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if req.Template == nil {
		return nil, fmt.Errorf("workflow template was not found in the request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	wts.setCreator(ctx, req.Template)
	return wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Create(req.Template)

}

func (c *WorkflowTemplateServer) setCreator(ctx context.Context, obj metav1.Object) {
	resource.Label(obj, common.LabelKeyControllerCreator, auth.GetUser(ctx).Name)
}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Get(req.Name, metav1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return wfTmpl, err
}

func (wts *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateListRequest) (*v1alpha1.WorkflowTemplateList, error) {
	wfClient := auth.GetWfClient(ctx)
	options := metav1.ListOptions{}
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

func (wts *WorkflowTemplateServer) DeleteWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateDeleteRequest) (*workflowtemplatepkg.WorkflowTemplateDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)

	err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace).Delete(req.Name, &metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &workflowtemplatepkg.WorkflowTemplateDeleteResponse{}, nil
}

func (wts *WorkflowTemplateServer) LintWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateLintRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(req.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, req.Template)
	if err != nil {
		return nil, err
	}

	return req.Template, nil
}

func (wts *WorkflowTemplateServer) UpdateWorkflowTemplate(ctx context.Context, req *workflowtemplatepkg.WorkflowTemplateUpdateRequest) (*v1alpha1.WorkflowTemplate, error) {
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
