package v1alpha1

func NamespaceOrDefault(namespace, defaultNamespace string) string {
	if namespace != "" {
		return namespace
	}
	return defaultNamespace
}

func NamespaceDefaultToEmpty(namespace, defaultNamespace string) string {
	if namespace == defaultNamespace {
		return ""
	}
	return namespace
}
