package workflowtemplate

import (
	"context"
	"fmt"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	commonserver "github.com/argoproj/argo/cmd/server/common"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/templateresolution"
	"github.com/argoproj/argo/workflow/validate"
)

type WorkflowTemplateServer struct {
	*commonserver.Server
}



func NewWorkflowTemplateServer(namespace string, wfClientset versioned.Interface, kubeClientSet kubernetes.Interface, enableClientAuth bool) *WorkflowTemplateServer {
	return &WorkflowTemplateServer{Server: commonserver.NewServer(enableClientAuth, namespace, wfClientset, kubeClientSet)}
}

func (wts *WorkflowTemplateServer) CreateWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}
	if wftmplReq.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace))

	err = validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, fmt.Errorf("Failed to create workflow template: %v", err)
	}

	return wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Create(wftmplReq.Template)

}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Get(wftmplReq.TemplateName, v1.GetOptions{})

	if err != nil {
		return nil, err
	}

	return wfTmpl, err
}

func (wts *WorkflowTemplateServer) ListWorkflowTemplates(ctx context.Context, wftmplReq *WorkflowTemplateListRequest) (*v1alpha1.WorkflowTemplateList, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}


	wfList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).List(v1.ListOptions{})

	if err != nil {
		return nil, err
	}

	return wfList, nil
}

func (wts *WorkflowTemplateServer) DeleteWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateDeleteRequest) (*WorkflowDeleteResponse, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}


	err = wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Delete(wftmplReq.TemplateName, &v1.DeleteOptions{})
	if err != nil {
		return nil, err
	}

	return &WorkflowDeleteResponse{
		TemplateName: wftmplReq.TemplateName,
		Status:       "Deleted",
	}, nil
}

func (wts *WorkflowTemplateServer) LintWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateCreateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace))

	err = validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, err
	}

	return wftmplReq.Template, nil
}


func (wts *WorkflowTemplateServer) UpdateWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateUpdateRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}
	if wftmplReq.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace))

	err = validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, fmt.Errorf("Failed to create workflow template: %v", err)
	}

	return wfClient.ArgoprojV1alpha1().WorkflowTemplates(wftmplReq.Namespace).Update(wftmplReq.Template)
}