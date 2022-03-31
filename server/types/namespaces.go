package types

type NamespacedRequest interface {
	GetNamespace() string
}

func Namespace(req interface{}) string {
	if v, ok := req.(NamespacedRequest); ok {
		return v.GetNamespace()
	}
	return ""
}
