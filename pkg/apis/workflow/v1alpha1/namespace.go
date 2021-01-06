package v1alpha1

func NamespaceOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func NamespaceIfDiff(a, b string) string {
	if a == b {
		return ""
	}
	return a
}
