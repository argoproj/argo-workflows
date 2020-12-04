package controller

type clusterName = string

const (
	// Represent all clusters, much like corev1.NamespaceAll.
	//  clusterAll     clusterName = ""
	defaultClusterName clusterName = "default"
)

// Return the clusterName, or the empty string if the clusterName == defaultClusterName.
// This will never return defaultClusterName.
func clusterNameDefaultAsEmpty(n clusterName) clusterName {
	if n != defaultClusterName {
		return n
	}
	return ""
}

// Return the clusterName, or defaultClusterName if the clusterName was empty
// This will never return an empty name.
func clusterNameOrDefault(n clusterName) clusterName {
	if n != "" {
		return n
	}
	return defaultClusterName
}
