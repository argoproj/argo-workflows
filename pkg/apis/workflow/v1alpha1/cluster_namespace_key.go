package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// controller-level unique key for a cluster's namespace
type ClusterNamespaceKey string

func ParseClusterNamespaceKey(s string) (ClusterNamespaceKey, error) {
	x := ClusterNamespaceKey(s)
	clusterName, gvr, _ := x.Split()
	if clusterName == "" || gvr.Empty() { // TODO - validate more
		return "nil", fmt.Errorf("must be 4 dot-delimited: \"clusterName.group.version.resource.namespace\", e.g. \"main.v1.pods.argo\"; only namespace maybe empty string")
	}
	return x, nil
}

func NewClusterNamespaceKey(clusterName ClusterName, gvr schema.GroupVersionResource, namespace string) ClusterNamespaceKey {
	return ClusterNamespaceKey(fmt.Sprintf("%v.%s.%s.%s.%s", clusterName, gvr.Group, gvr.Version, gvr.Resource, namespace))
}

func (x ClusterNamespaceKey) Split() (clusterName ClusterName, gvr schema.GroupVersionResource, namespace string) {
	parts := strings.Split(string(x), ".")
	if len(parts) != 5 {
		return "", schema.GroupVersionResource{}, ""
	}
	return ClusterName(parts[0]), schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}, parts[4]
}
