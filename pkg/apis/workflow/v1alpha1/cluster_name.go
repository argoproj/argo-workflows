package v1alpha1

type ClusterName string

func ClusterNameOr(a ClusterName, b ClusterName) ClusterName {
	if a != "" {
		return a
	}
	return b
}

func ClusterNameIfNot(a, b ClusterName) ClusterName {
	if a == b {
		return ""
	}
	return a
}
