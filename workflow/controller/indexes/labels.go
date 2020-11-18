package indexes

import (
	"fmt"

	"github.com/prometheus/common/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/workflow/util"
)

func MetaNamespaceLabelIndex(namespace, label string) string {
	return namespace + "/" + label
}

func MetaLabelIndexFunc(label string) cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return []string{}, fmt.Errorf("object has no meta: %v", err)
		}
		if value, exists := v.GetLabels()[label]; exists {
			return []string{value}, nil
		} else {
			return []string{}, nil
		}
	}
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

func MetaNamespaceLabelIndexFunc(label string) cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return []string{}, fmt.Errorf("object has no meta: %v", err)
		}
		if value, exists := v.GetLabels()[label]; exists {
			return []string{MetaNamespaceLabelIndex(v.GetNamespace(), value)}, nil
		} else {
			return []string{}, nil
		}
	}
}
