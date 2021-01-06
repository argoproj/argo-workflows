package v1alpha1

import (
	"fmt"
	"strings"
)

// controller-level unique key for a cluster's namespace
type ClusterNamespaceKey string

func ParseClusterNamespaceKey(s string) (ClusterNamespaceKey, error) {
	if !strings.Contains(s, ".") {
		return "", fmt.Errorf("dot delimiter missing in '%s', do you mean '%s.'?", s, s)
	}
	return ClusterNamespaceKey(s), nil
}

func NewClusterNamespaceKey(clusterName ClusterName, namespace string) ClusterNamespaceKey {
	return ClusterNamespaceKey(fmt.Sprintf("%v.%s", clusterName, namespace))
}

func (x ClusterNamespaceKey) Split() (clusterName ClusterName, namespace string) {
	parts := strings.Split(string(x), ".")
	if len(parts) != 2 {
		return "", ""
	}
	return ClusterName(parts[0]), parts[1]
}
