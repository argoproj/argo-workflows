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
	namespace := wts.Namespace
	if wftmplReq.Namespace != "" {
		namespace = wftmplReq.Namespace
	}
	if wftmplReq.Template == nil {
		return nil, fmt.Errorf("WorkflowTemplate is not found in Request body")
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))

	err = validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, fmt.Errorf("Failed to create workflow template: %v", err)
	}

	created, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace).Create(wftmplReq.Template)

	if err != nil {
		return nil, err
	}

	return created, err
}

func (wts *WorkflowTemplateServer) GetWorkflowTemplate(ctx context.Context, wftmplReq *WorkflowTemplateGetRequest) (*v1alpha1.WorkflowTemplate, error) {
	wfClient, _, err := wts.GetWFClient(ctx)
	if err != nil {
		return nil, err
	}

	namespace := wts.Namespace
	if wftmplReq.Namespace != "" {
		namespace = wftmplReq.Namespace
	}

	wfTmpl, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace).Get(wftmplReq.TemplateName, v1.GetOptions{})

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

	namespace := wts.Namespace
	if wftmplReq.Namespace != "" {
		namespace = wftmplReq.Namespace
	}

	wfList, err := wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace).List(v1.ListOptions{})

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

	namespace := wts.Namespace
	if wftmplReq.Namespace != "" {
		namespace = wftmplReq.Namespace
	}

	err = wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace).Delete(wftmplReq.TemplateName, &v1.DeleteOptions{})
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

	namespace := wts.Namespace
	if wftmplReq.Namespace != "" {
		namespace = wftmplReq.Namespace
	}
	wftmplGetter := templateresolution.WrapWorkflowTemplateInterface(wfClient.ArgoprojV1alpha1().WorkflowTemplates(namespace))

	err = validate.ValidateWorkflowTemplate(wftmplGetter, wftmplReq.Template)
	if err != nil {
		return nil, err
	}

	return wftmplReq.Template, nil
}
