package templateresolution

import (
	"github.com/argoproj/argo/pkg/apis/workflow"
	v1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// lazyWorkflowTemplate retrieves WorkflowTemplate lazily.
type lazyWorkflowTemplate struct {
	// wftmplGetter is a proxied WorkflowTemplate getter.
	wftmplGetter WorkflowTemplateNamespacedGetter
	// wftmpl is a cache of retrieved WorkflowTemplate.
	wftmpl *wfv1.WorkflowTemplate
	// namespace is the namespace of the WorkflowTemplate.
	namespace string
	// name is the name of the WorkflowTemplate.
	name string
}

var _ wfv1.TemplateGetter = &lazyWorkflowTemplate{}

// NewLazyWorkflowTemplate is a public constructor of lazyWorkflowTemplate.
func NewLazyWorkflowTemplate(wftmplGetter WorkflowTemplateNamespacedGetter, namespace, name string) *lazyWorkflowTemplate {
	return &lazyWorkflowTemplate{
		wftmplGetter: wftmplGetter,
		namespace:    namespace,
		name:         name,
	}
}

// GetNamespace returns the namespace of the WorkflowTemplate.
func (lwt *lazyWorkflowTemplate) GetNamespace() string {
	return lwt.namespace
}

// GetName returns the name of the WorkflowTemplate.
func (lwt *lazyWorkflowTemplate) GetName() string {
	return lwt.name
}

// GroupVersionKind returns a GroupVersionKind of WorkflowTemplate.
func (lwt *lazyWorkflowTemplate) GroupVersionKind() schema.GroupVersionKind {
	return v1alpha1.SchemeGroupVersion.WithKind(workflow.WorkflowTemplateKind)
}

// GetTemplateByName retrieves a defined template by its name
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
