package indexes

import (
	corev1 "k8s.io/api/core/v1"
)

func PodPhaseIndexFunc(obj any) ([]string, error) {
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		return nil, nil
	}
	return []string{string(pod.Status.Phase)}, nil
}
