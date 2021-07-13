package informer

import (
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTaskSetLister struct {
	delegate cache.GenericLister
}

var _ v1alpha1.WorkflowTaskSetLister = &tolerantWorkflowTaskSetLister{}

func (t *tolerantWorkflowTaskSetLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTaskSet, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTaskSets(list), err
}

func (t *tolerantWorkflowTaskSetLister) WorkflowTaskSets(namespace string) v1alpha1.WorkflowTaskSetNamespaceLister {
	return &tolerantWorkflowTaskSetNamespaceLister{delegate: t.delegate.ByNamespace(namespace)}
}
