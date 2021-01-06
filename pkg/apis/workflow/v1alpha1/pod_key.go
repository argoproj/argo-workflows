package v1alpha1

import (
	"fmt"
	"strings"
)

// controller-level unique key for a pod
type PodKey string

func NewPodKey(clusterName ClusterName, namespace, podName string) PodKey {
	return PodKey(fmt.Sprintf("%s/%s/%s", clusterName, namespace, podName))
}

func (key PodKey) Split() (clusterName ClusterName, namespace string, name string) {
	parts := strings.Split(string(key), "/")
	if len(parts) != 3 {
		return "", "", ""
	}
	return ClusterName(parts[0]), parts[1], parts[2]
}
