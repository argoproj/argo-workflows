package types

type NamespacedRequest interface {
	GetNamespace() string
}

type NamespaceHolder struct {
	namespace string
}

func NewNamespaceHolder(namespace string) *NamespaceHolder {
	return &NamespaceHolder{
		namespace: namespace,
	}
}

func (n *NamespaceHolder) GetNamespace() string {
	return n.namespace
}
