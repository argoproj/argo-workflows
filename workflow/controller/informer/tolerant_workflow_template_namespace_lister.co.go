package informer

import (
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTemplateNamespaceLister struct {
	delegate cache.GenericNamespaceLister
}

var _ v1alpha1.WorkflowTemplateNamespaceLister = &tolerantWorkflowTemplateNamespaceLister{}

func (t *tolerantWorkflowTemplateNamespaceLister) Get(name string) (*wfv1.WorkflowTemplate, error) {
	object, err := t.delegate.Get(name)
	if err != nil {
		return nil, err
	}
	return objectToWorkflowTemplate(object)
}

func (t *tolerantWorkflowTemplateNamespaceLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTemplate, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTemplates(list), err
}
