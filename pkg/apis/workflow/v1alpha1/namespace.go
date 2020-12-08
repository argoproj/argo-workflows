package v1alpha1

func NamespaceOrOther(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func NamespaceOtherAsEmpty(a, b string) string {
	if a == b {
		return ""
	}
	return a
}
