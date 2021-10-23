package types

type NamespacedRequest interface {
	GetNamespace() string
}

type NamespaceHolder struct {
	namespace string
}

func NewNamespaceHolder(namepsace string) *NamespaceHolder {
	return &NamespaceHolder{
		namespace: namepsace,
	}
}

func (n *NamespaceHolder) GetNamespace() string {
	return n.namespace
}
