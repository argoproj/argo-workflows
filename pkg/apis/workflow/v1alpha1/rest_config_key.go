package v1alpha1

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// controller-level unique key for a cluster's namespace
type RestConfigKey string

func ParseRestConfigKey(s string) (RestConfigKey, error) {
	x := RestConfigKey(s)
	clusterName, gvr, _ := x.Split()
	if clusterName == "" || gvr.Empty() { // TODO - validate more
		return "nil", fmt.Errorf("must be dot-delimited: \"clusterName.group.version.resource.namespace\", e.g. \"main.v1.pods.argo\"; only namespace maybe empty string: %s", s)
	}
	return x, nil
}

func NewRestConfigKey(clusterName ClusterName, gvr schema.GroupVersionResource, namespace string) RestConfigKey {
	return RestConfigKey(fmt.Sprintf("%v.%s.%s.%s.%s", clusterName, gvr.Group, gvr.Version, gvr.Resource, namespace))
}

func (x RestConfigKey) Split() (clusterName ClusterName, gvr schema.GroupVersionResource, namespace string) {
	parts := strings.Split(string(x), ".")
	if len(parts) != 5 {
		return "", schema.GroupVersionResource{}, ""
	}
	return ClusterName(parts[0]), schema.GroupVersionResource{Group: parts[1], Version: parts[2], Resource: parts[3]}, parts[4]
}
