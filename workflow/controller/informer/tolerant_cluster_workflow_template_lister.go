package informer

import (
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type tolerantClusterWorkflowTemplateLister struct {
	delegate cache.GenericLister
}

func (t *tolerantClusterWorkflowTemplateLister) List(selector v1Label.Selector) ([]*wfv1.ClusterWorkflowTemplate, error) {
	list, err := t.delegate.List(selector)
	return objectsToClusterWorkflowTemplates(list), err
}

func (t *tolerantClusterWorkflowTemplateLister) Get(name string) (*wfv1.ClusterWorkflowTemplate, error) {
	object, err := t.delegate.Get(name)
	if err != nil {
		return nil, err
	}
	return objectToClusterWorkflowTemplate(object)
}
