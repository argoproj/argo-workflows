package informer

import (
	"fmt"
	"reflect"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func objectToWorkflowTemplate(object runtime.Object) (*wfv1.WorkflowTemplate, error) {
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("malformed workflow template: expected *unstructured.Unstructured, got %s", reflect.TypeOf(object).Name())
	}
	v := &wfv1.WorkflowTemplate{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	if err != nil {
		return nil, fmt.Errorf("malformed workflow template %s/%s: %w", un.GetNamespace(), un.GetName(), err)
	}
	return v, nil
}

func objectsToWorkflowTemplates(list []runtime.Object) []*wfv1.WorkflowTemplate {
	ret := make([]*wfv1.WorkflowTemplate, 0)
	for _, object := range list {
		v, err := objectToWorkflowTemplate(object)
		if err != nil {
			log.Error(err)
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
