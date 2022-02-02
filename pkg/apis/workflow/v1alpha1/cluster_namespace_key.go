package v1alpha1

import (
	"fmt"
	"strings"
)

var (
	// ErrInvalidClusterNamespaceKey is kicked when the clusterName.namespace
	// format is not followed
	ErrInvalidClusterNamespaceKey string = "invalid cluster namespace key: must be dot-delimited: \"clusterName.namespace\", e.g. \"main.argo\"; only namespace maybe empty string: %s"
	// ErrIncompleteClusterNamespaceKey is kicked when clusterName or namespace is
	// empty
	ErrIncompleteClusterNamespaceKey string = "incomplete cluster namespace key: need a clusterName and a namespace but got clusterName: %s and namespace %s"
)

// ClusterNamespaceKey is a controller-level unique key for a
// cluster's namespace
type ClusterNamespaceKey string

// ParseClusterNamespaceKey
func ParseClusterNamespaceKey(s string) (ClusterNamespaceKey, error) {
	key := ClusterNamespaceKey(s)
	_, _, err := key.Split()
	if err != nil {
		return key, err
	}

	return key, nil
}

// NewClusterNamespaceKey creates a ClusterNamespaceKey from a cluster name and
// a namespace
func NewClusterNamespaceKey(clusterName ClusterName, namespace string) (ClusterNamespaceKey, error) {
	key := ClusterNamespaceKey(fmt.Sprintf("%v.%s", clusterName, namespace))
	return key, key.Validate()
}

// Split takes a ClusterNamespaceKey from the clusterName.namespace form and
// gives us the clusterName and the namespace separated
func (key *ClusterNamespaceKey) Split() (ClusterName, string, error) {
	parts := strings.Split(key.String(), ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf(ErrInvalidClusterNamespaceKey, key.String())
	}

	return ClusterName(parts[0]), parts[1], nil
}

// String gives a stringified version of a ClusterNamespaceKey
func (key ClusterNamespaceKey) String() string {
	return string(key)
}

// Validate checks whether a cluster namespace key conforms to the
// clusterName.namespace standard
func (key *ClusterNamespaceKey) Validate() error {
	parts := strings.Split(key.String(), ".")
	if len(parts) != 2 {
		return fmt.Errorf(ErrInvalidClusterNamespaceKey, key.String())
	}

	clusterName := parts[0]
	namespace := parts[1]
	if clusterName == "" || namespace == "" {
		return fmt.Errorf(ErrIncompleteClusterNamespaceKey, clusterName, namespace)
	}

	return nil
}
