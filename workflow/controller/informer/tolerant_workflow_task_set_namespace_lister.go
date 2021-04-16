package informer

import (
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/listers/workflow/v1alpha1"
)

type tolerantWorkflowTaskSetNamespaceLister struct {
	delegate cache.GenericNamespaceLister
}

var _ v1alpha1.WorkflowTaskSetNamespaceLister = &tolerantWorkflowTaskSetNamespaceLister{}

func (t *tolerantWorkflowTaskSetNamespaceLister) Get(name string) (*wfv1.WorkflowTaskSet, error) {
	object, err := t.delegate.Get(name)
	if err != nil {
		return nil, err
	}
	return objectToWorkflowTaskSet(object)
}

func (t *tolerantWorkflowTaskSetNamespaceLister) List(selector v1Label.Selector) ([]*wfv1.WorkflowTaskSet, error) {
	list, err := t.delegate.List(selector)
	return objectsToWorkflowTaskSets(list), err
}
