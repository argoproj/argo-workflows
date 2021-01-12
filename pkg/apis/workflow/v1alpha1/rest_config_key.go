package v1alpha1

import (
	"fmt"
	"strings"
)

// controller-level unique key for a cluster's namespace
type ClusterNamespaceKey string

func ParseClusterNamespaceKey(s string) (ClusterNamespaceKey, error) {
	x := ClusterNamespaceKey(s)
	clusterName, _ := x.Split()
	if clusterName == "" { // TODO - validate more
		return "nil", fmt.Errorf("must be dot-delimited: \"clusterName.namespace\", e.g. \"main.argo\"; only namespace maybe empty string: %s", s)
	}
	return x, nil
}

func NewClusterNamespaceKey(clusterName ClusterName, namespace string) ClusterNamespaceKey {
	return ClusterNamespaceKey(fmt.Sprintf("%v.%s", clusterName, namespace))
}

func (x ClusterNamespaceKey) Split() (clusterName ClusterName, namespace string) {
	parts := strings.Split(string(x), ".")
	if len(parts) != 5 {
		return "", ""
	}
	return ClusterName(parts[0]), parts[1]
}
