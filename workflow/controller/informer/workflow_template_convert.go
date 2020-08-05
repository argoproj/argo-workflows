package informer

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func objectToWorkflowTemplate(object runtime.Object) (*wfv1.WorkflowTemplate, error) {
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to convert workflow template object to unstructured")
	}
	v := &wfv1.WorkflowTemplate{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	return v, err
}

func objectsToWorkflowTemplates(list []runtime.Object) []*wfv1.WorkflowTemplate {
	ret := make([]*wfv1.WorkflowTemplate, 0)
	for _, object := range list {
		v, err := objectToWorkflowTemplate(object)
		if err != nil {
			log.WithError(err).Error("failed convert unstructured workflow template")
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
