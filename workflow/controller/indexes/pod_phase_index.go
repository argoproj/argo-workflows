package indexes

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func PodPhaseIndexFunc() cache.IndexFunc {
	return func(obj interface{}) ([]string, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return []string{}, nil
		}
		return []string{string(pod.Status.Phase)}, nil
	}
}
