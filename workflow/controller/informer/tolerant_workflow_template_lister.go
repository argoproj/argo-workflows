package informer

import (
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTemplateLister struct {
	delegate cache.GenericLister
}

var _ v1alpha1.WorkflowTemplateLister = &tolerantWorkflowTemplateLister{}

func (t *tolerantWorkflowTemplateLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTemplate, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTemplates(list), err
}

func (t *tolerantWorkflowTemplateLister) WorkflowTemplates(namespace string) v1alpha1.WorkflowTemplateNamespaceLister {
	return &tolerantWorkflowTemplateNamespaceLister{delegate: t.delegate.ByNamespace(namespace)}
}
