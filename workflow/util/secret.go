package util

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func SecretFromUnstructured(un *unstructured.Unstructured) (*apiv1.Secret, error) {
	serviceAccount := &apiv1.Secret{}
	return serviceAccount, runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, serviceAccount)
}

func SecretToUnstructured(serviceAccount *apiv1.Secret ) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(serviceAccount)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
