package podlister

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// this takes a list, e.g. from cache.SharedIndexInformer, and converts it to pods
func Pods(list []interface{}) ([]corev1.Pod, error) {
	pods := make([]corev1.Pod, len(list))
	for i, item := range list {
		var pod corev1.Pod
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.(*unstructured.Unstructured).Object, &pod)
		if err != nil {
			return nil, err
		}
		pods[i] = pod
	}
	return pods, nil
}
