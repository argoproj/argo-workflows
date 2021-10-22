package types

type NamespacedRequest interface {
    GetNamespace() string
}

type NamespaceContainer struct {
    namespace string
}

func NewNamespaceContainer(namepsace string) *NamespaceContainer {
    return &NamespaceContainer{
        namespace: namepsace,
    }
}

func (n *NamespaceContainer) GetNamespace() string {
    return n.namespace
}
