package indexes

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
)

func MetaNamespaceLabelIndex(namespace, label string) string {
	return namespace + "/" + label
}

func MetaWorkflowPhaseIndexFunc() cache.IndexFunc {
	return func(obj any) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		if value, exists := v.GetLabels()[common.LabelKeyPhase]; exists {
			return []string{value}, nil
		}
		// If the object doesn't have a phase set, consider it pending
		return []string{string(v1alpha1.NodePending)}, nil
	}
}

func MetaNamespaceLabelIndexFunc(label string) cache.IndexFunc {
	return func(obj any) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		if value, exists := v.GetLabels()[label]; exists {
			return []string{MetaNamespaceLabelIndex(v.GetNamespace(), value)}, nil
		}
		return nil, nil
	}
}
