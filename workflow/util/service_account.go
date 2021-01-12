package util

import (
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func ServiceAccountFromUnstructured(un *unstructured.Unstructured) (*apiv1.ServiceAccount, error) {
	serviceAccount := &apiv1.ServiceAccount{}
	return serviceAccount, runtime.DefaultUnstructuredConverter.FromUnstructured(un.Object, serviceAccount)
}

func ServiceAccountToUnstructured(serviceAccount *apiv1.ServiceAccount, ) (*unstructured.Unstructured, error) {
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(serviceAccount)
	if err != nil {
		return nil, err
	}
	return &unstructured.Unstructured{Object: obj}, nil
}
