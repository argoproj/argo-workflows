package v1alpha1

type ClusterName string

func ClusterNameOrOther(a ClusterName, b ClusterName) ClusterName {
	if a != "" {
		return a
	}
	return b
}

func ClusterNameOtherAsEmpty(a, b ClusterName) ClusterName {
	if a == b {
		return ""
	}
	return a
}
