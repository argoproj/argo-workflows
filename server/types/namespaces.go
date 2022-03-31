package types

type NamespacedRequest interface {
	GetNamespace() string
}

func Namespace(m interface{}) string {
	if v, ok := m.(NamespacedRequest); ok {
		return v.GetNamespace()
	}
	return ""
}
