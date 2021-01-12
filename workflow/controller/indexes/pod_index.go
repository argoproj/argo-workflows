package indexes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func PodPhaseIndexFunc(obj interface{}) ([]string, error) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, nil
	}
	value, _, _ := unstructured.NestedString(un.Object, "status", "phase")
	return []string{value}, nil
}
