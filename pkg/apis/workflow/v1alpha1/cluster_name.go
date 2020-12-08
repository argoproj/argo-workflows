package v1alpha1

type ClusterName = string

const (
	// Represent all clusters, much like corev1.NamespaceAll.
	ClusterAll  ClusterName = ""
	ThisCluster ClusterName = "."
)

// Return the clusterName, or ThisCluster if the clusterName was empty
// This will never return an empty name.
func ClusterNameOrThis(n ClusterName) ClusterName {
	if n != "" {
		return n
	}
	return ThisCluster
}
