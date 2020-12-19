package v1alpha1

type ClusterName = string

const (
	// Represent all clusters, much like corev1.NamespaceAll.
	ClusterAll ClusterName = ""
)

func ClusterNameOrOther(a ClusterName, b ClusterName) ClusterName {
	if a != "" {
		return a
	}
	return b
}
