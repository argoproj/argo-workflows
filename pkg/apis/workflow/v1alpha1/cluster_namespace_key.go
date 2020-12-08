package v1alpha1

// controller-level uniquely key for a cluster's namespace
type ClusterNamespaceKey = string

func NewClusterNamespaceKey(clusterName ClusterName, namespace string) ClusterNamespaceKey {
	return ClusterNameOrThis(clusterName) + "/" + namespace
}
