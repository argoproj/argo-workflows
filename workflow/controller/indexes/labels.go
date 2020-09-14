package indexes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/cache"
)

func MetaNamespaceLabelIndex(namespace, label string) string {
	return namespace + "/" + label
}

func MetaNamespaceLabelIndexFunc(label string) cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		v, err := meta.Accessor(obj)
		if err != nil {
			return []string{}, fmt.Errorf("object has no meta: %v", err)
		}
		return []string{MetaNamespaceLabelIndex(v.GetNamespace(), v.GetLabels()[label])}, nil
	}
}
