package v1alpha1

import (
	"fmt"
	"strings"
)

// controller-level unique key for a pod
type PodKey = string

func NewPodKey(clusterName ClusterName, namespace, podName string) PodKey {
	return fmt.Sprintf("%s/%s/%s", ClusterNameOrThis(clusterName), namespace, podName)
}

func SplitPodKey(key PodKey) (clusterName ClusterName, namespace string, name string) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", "", ""
	}
	return parts[0], parts[1], parts[2]
}
