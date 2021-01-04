package indexes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func MetaNamespaceLabelIndex(namespace, label string) string {
	return namespace + "/" + label
}

func MetaWorkflowPhaseIndexFunc() cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return []string{}, fmt.Errorf("object has no meta: %v", err)
		}
		if value, exists := v.GetLabels()[common.LabelKeyPhase]; exists {
			return []string{value}, nil
		} else {
			// If the object doesn't have a phase set, consider it pending
			return []string{string(v1alpha1.NodePending)}, nil
		}
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
