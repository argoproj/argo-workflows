package v1alpha1

type ClusterName = string

const (
	// Represent all clusters, much like corev1.NamespaceAll.
	ClusterAll         ClusterName = ""
	DefaultClusterName ClusterName = "default"
)

// Return the clusterName, or DefaultClusterName if the clusterName was empty
// This will never return an empty name.
func ClusterNameOrDefault(n ClusterName) ClusterName {
	if n != "" {
		return n
	}
	return DefaultClusterName
}
