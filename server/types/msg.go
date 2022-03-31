package types

type Msg struct {
	Cluster   string
	Namespace string
	Act       string
	Resource  string
}

func (n *Msg) GetNamespace() string {
	return n.Namespace
}

func (n *Msg) GetCluster() string {
	return n.Cluster
}

func (n *Msg) GetResource() string {
	return n.Resource
}

func (n *Msg) GetAct() string {
	return n.Act
}
