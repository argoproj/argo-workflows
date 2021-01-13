package util

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func SecretToUnstructured(serviceAccount *apiv1.Secret) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(serviceAccount)
	if err != nil {
		return nil, err
	}
	un := &unstructured.Unstructured{Object: obj}
	un.SetKind("Secret")
	un.SetAPIVersion("v1")
	return un, nil
}
