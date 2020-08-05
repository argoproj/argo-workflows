package informer

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func objectToClusterWorkflowTemplate(object runtime.Object) (*wfv1.ClusterWorkflowTemplate, error) {
	un, ok := object.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("failed to convert cluster workflow template object to unstructured")
	}
	v := &wfv1.ClusterWorkflowTemplate{}
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, v)
	return v, err
}

func objectsToClusterWorkflowTemplates(list []runtime.Object) []*wfv1.ClusterWorkflowTemplate {
	ret := make([]*wfv1.ClusterWorkflowTemplate, 0)
	for _, object := range list {
		v, err := objectToClusterWorkflowTemplate(object)
		if err != nil {
			log.WithError(err).Error("failed convert unstructured cluster workflow template")
			continue
		}
		ret = append(ret, v)
	}
	return ret
}
