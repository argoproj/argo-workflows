package informer

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func objectToWorkflowTemplate(object runtime.Object) (*wfv1.WorkflowTemplate, error) {
	v := &wfv1.WorkflowTemplate{}
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return v, fmt.Errorf("malformed workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	if err != nil {
		return v, fmt.Errorf("malformed workflow template \"%s/%s\": %w", un.GetNamespace(), un.GetName(), err)
	}
	return v, nil
}

func objectsToWorkflowTemplates(list []runtime.Object) []*wfv1.WorkflowTemplate {
	ret := make([]*wfv1.WorkflowTemplate, len(list))
	for i, object := range list {
		ret[i], _ = objectToWorkflowTemplate(object)
	}
	return ret
}
