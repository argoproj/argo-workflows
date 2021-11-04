package types

type NamespacedRequest interface {
	GetNamespace() string
}

type NamespaceHolder string

func (n NamespaceHolder) GetNamespace() string {
	return string(n)
}
