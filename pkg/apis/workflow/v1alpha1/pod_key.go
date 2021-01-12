package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// controller-level unique key for a pod
type PodKey string

func NewPodKey(clusterName ClusterName, gvr schema.GroupVersionResource, namespace, podName string) PodKey {
	return PodKey(fmt.Sprintf("%s/%s/%s/%s/%s/%s", clusterName, gvr.Group, gvr.Version, gvr.Resource, namespace, podName))
}

func (key PodKey) Split() (clusterName ClusterName, gvr schema.GroupVersionResource, namespace string, name string) {
	parts := strings.Split(string(key), "/")
	if len(parts) != 6 {
		return "", schema.GroupVersionResource{}, "", ""
	}
	return ClusterName(parts[0]), schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}, parts[4], parts[5]
}
