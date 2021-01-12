package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// controller-level unique key for a resource
type ResourceKey string

func NewResourceKey(clusterName ClusterName, gvr schema.GroupVersionResource, namespace, name string) ResourceKey {
	return ResourceKey(fmt.Sprintf("%s/%s/%s/%s/%s/%s", clusterName, gvr.Group, gvr.Version, gvr.Resource, namespace, name))
}

func (key ResourceKey) Split() (clusterName ClusterName, gvr schema.GroupVersionResource, namespace string, name string) {
	parts := strings.Split(string(key), "/")
	if len(parts) != 6 {
		return "", schema.GroupVersionResource{}, "", ""
	}
	return ClusterName(parts[0]), schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}, parts[4], parts[5]
}
