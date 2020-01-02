package workflowtemplate

import (
	"context"
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/errors"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo/cmd/server/auth"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type WorkflowTemplateServer struct {
}

func NewWorkflowTemplateServer() WorkflowTemplateServiceServer {
	return &WorkflowTemplateServer{}
}

func (wts *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	if wftmplReq.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, fmt.Errorf("Failed to create workflow template: %v", err)
	}

	return wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Create(wftmplReq.Template)

}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Get(wftmplReq.TemplateName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return wfTmpl, err
}

func (wts *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, wftmplReq *WorkflowTemplateListRequest) (*v1alpha1.WorkflowTemplateList, error) {
	wfClient := auth.GetWfClient(ctx)

	wfList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return wfList, nil
}

func (wts *WorkflowTemplateServer) DeleteWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient := auth.GetWfClient(ctx)

	err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Delete(wftmplReq.TemplateName, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &WorkflowDeleteResponse{
		TemplateName: wftmplReq.TemplateName,
		Status:       "Deleted",
	}, nil
}

func (wts *WorkflowTemplateServer) LintWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, err
	}

	return wftmplReq.Template, nil
}

func (wts *WorkflowTemplateServer) UpdateWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateUpdateRequest) (*v1alpha1.WorkflowTemplate, error) {
	if wftmplReq.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	return wts.updateWorkflowTemplate(ctx, wftmplReq.Template)
}

func (wts *WorkflowTemplateServer) UpdateWorkflowTemplateSpec(ctx context.Context, wftmplReq *WorkflowTemplateSpecUpdateRequest) (*v1alpha1.WorkflowTemplateSpec, error) {
	wfClient := auth.GetWfClient(ctx)

	if wftmplReq.TemplateSpec == nil {
		return nil, fmt.Errorf("WorkflowTemplate spec is not found in Request body")
	}

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Get(wftmplReq.TemplateName, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template with name '%s': %v", wftmplReq.TemplateName, err)
	}

	wfTmpl.Spec = *wftmplReq.TemplateSpec
	newWfTmpl, err := wts.updateWorkflowTemplate(ctx, wfTmpl)
	if err != nil {
		return nil, err
	}
	return &newWfTmpl.Spec, nil
}

func (wts *WorkflowTemplateServer) updateWorkflowTemplate(ctx context.Context, newWfTmpl *v1alpha1.WorkflowTemplate) (*v1alpha1.WorkflowTemplate, error) {
	wfClient := auth.GetWfClient(ctx)
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(newWfTmpl.Namespace))

	err := validate.ValidateWorkflowTemplate(wftmplGetter, newWfTmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow template: %v", err)
	}

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(newWfTmpl.Namespace).Get(newWfTmpl.Name, v1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow template with name '%s': %v", newWfTmpl.Name, err)
	}

	for i := 0; i < 10; i++ {
		wfTmpl.Spec = newWfTmpl.Spec
		wfTmpl.Labels = newWfTmpl.Labels
		wfTmpl.Annotations = newWfTmpl.Annotations

		res, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(newWfTmpl.Namespace).Update(wfTmpl)
		if err == nil {
			return res, nil
		}
		if !errors.IsConflict(err) {
			return nil, err
		}

		wfTmpl, err = wfClient.ArgoprojV1alpha1().WorkflowTemplates(newWfTmpl.Namespace).Get(newWfTmpl.Name, v1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get workflow template with name '%s': %v", newWfTmpl.Name, err)
		}
	}
	return nil, status.Errorf(codes.Internal, "Failed to update application. Too many conflicts")
}

