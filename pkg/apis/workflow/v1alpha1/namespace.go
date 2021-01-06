package v1alpha1

func NamespaceOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func NamespaceIfNot(a, b string) string {
	if a == b {
		return ""
	}
	return a
}
