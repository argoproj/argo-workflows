package templateresolution

import (
	"github.com/argoproj/argo/pkg/apis/workflow"
	v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type lazyWorkflowTemplate struct {
	wftmplGetter WorkflowTemplateNamespacedGetter
	wftmpl       *wfv1.WorkflowTemplate
	namespace    string
	name         string
}

var _ wfv1.TemplateGetter = &lazyWorkflowTemplate{}

func NewLazyWorkflowTemplate(wftmplGetter WorkflowTemplateNamespacedGetter, namespace, name string) *lazyWorkflowTemplate {
	return &lazyWorkflowTemplate{
		wftmplGetter: wftmplGetter,
		namespace:    namespace,
		name:         name,
	}
}

func (lwt *lazyWorkflowTemplate) GetNamespace() string {
	return lwt.namespace
}

func (lwt *lazyWorkflowTemplate) GetName() string {
	return lwt.name
}

func (lwt *lazyWorkflowTemplate) GroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind)
}

func (lwt *lazyWorkflowTemplate) GetTemplateByName(name string) *wfv1.Template {
	err := lwt.ensureWorkflowTemplate()
	if err != nil {
		return nil
	}
	return lwt.wftmpl.GetTemplateByName(name)
}

func (lwt *lazyWorkflowTemplate) ensureWorkflowTemplate() error {
	if lwt.wftmpl == nil {
		wftmpl, err := lwt.wftmplGetter.Get(lwt.name)
		if err != nil {
			return err
		}
		lwt.wftmpl = wftmpl
	}
	return nil
}
