package informer

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

// this function always tries to return a value, even if it is badly formed
func objectToClusterWorkflowTemplate(object runtime.Object) (*wfv1.ClusterWorkflowTemplate, error) {
	v := &wfv1.ClusterWorkflowTemplate{}
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return v, fmt.Errorf("malformed cluster workflow template: expected \"*unstructured.Unstructured\", got \"%s\"", reflect.TypeOf(object).String())
	}
	err := util.FromUnstructuredObj(un, v)
	if err != nil {
		return v, fmt.Errorf("malformed cluster workflow template \"%s\": %w", un.GetName(), err)
	}
	return v, nil
}

func objectsToClusterWorkflowTemplates(list []runtime.Object) []*wfv1.ClusterWorkflowTemplate {
	ret := make([]*wfv1.ClusterWorkflowTemplate, len(list))
	for i, object := range list {
		ret[i], _ = objectToClusterWorkflowTemplate(object)
	}
	return ret
}
