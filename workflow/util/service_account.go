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
