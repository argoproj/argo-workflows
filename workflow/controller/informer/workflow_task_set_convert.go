package informer

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

func objectToWorkflowTaskSet(object runtime.Object) (*wfv1.WorkflowTaskSet, error) {
	v := &wfv1.WorkflowTaskSet{}
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return v, fmt.Errorf("malformed workflow taskset: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	err := util.FromUnstructuredObj(un, v)
	if err != nil {
		return v, fmt.Errorf("malformed workflow taskset \"%s/%s\": %w", un.GetNamespace(), un.GetName(), err)
	}
	return v, nil
}

func objectsToWorkflowTaskSets(list []runtime.Object) []*wfv1.WorkflowTaskSet {
	ret := make([]*wfv1.WorkflowTaskSet, len(list))
	for i, object := range list {
		ret[i], _ = objectToWorkflowTaskSet(object)
	}
	return ret
}
