package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// controller-level unique key for a resource
type ResourceKey string

func NewResourceKey(clusterName ClusterName, namespace, name string, gvr schema.GroupVersionResource) ResourceKey {
	return ResourceKey(fmt.Sprintf("%s/%s/%s/%s.%s.%s", clusterName, namespace, name, gvr.Resource, gvr.Version, gvr.Group))
}

func (key ResourceKey) Split() (clusterName ClusterName, namespace, name string, gvr schema.GroupVersionResource) {
	parts := strings.Split(string(key), "/")
	if len(parts) != 4 {
		return "", "", "", schema.GroupVersionResource{}
	}
	x, _ := schema.ParseResourceArg(parts[3])
	if x.Empty() {
		return "", "", "", schema.GroupVersionResource{}
	}
	return ClusterName(parts[0]), parts[1], parts[2], *x
}
