package indexes

import (
	"fmt"

	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

func MetaWorkflowIndexFunc(obj interface{}) ([]string, error) {
	m, err := meta.Accessor(obj)
	if err != nil {
		return []string{}, fmt.Errorf("object has no meta: %v", err)
	}
	name, ok := m.GetLabels()[common.LabelKeyWorkflow]
	if !ok {
		return []string{}, fmt.Errorf("object has no workflow label")
	}
	namespace := wfv1.NamespaceOrOther(m.GetLabels()[common.LabelKeyWorkflowNamespace], m.GetNamespace())
	return []string{WorkflowIndexValue(namespace, name)}, nil
}

func WorkflowIndexValue(namespace, name string) string {
	return namespace + "/" + name
}

func WorkflowSemaphoreKeysIndexFunc() cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			log.Warnf("cannot convert obj into unstructured.Unstructured in Indexer %s", SemaphoreConfigIndexName)
			return []string{}, nil
		}
		wf, err := util.FromUnstructured(un)
		if err != nil {
			log.Warnf("failed to convert to workflow from unstructured: %v", err)
			return []string{}, nil
		}
		return wf.GetSemaphoreKeys(), nil
	}
}
