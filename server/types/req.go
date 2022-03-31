package types

type Req struct {
	Cluster   string
	Namespace string
	Act       string
	Resource  string
}

func (r *Req) GetNamespace() string {
	return r.Namespace
}

func (r *Req) GetCluster() string {
	return r.Cluster
}

func (r *Req) GetResource() string {
	return r.Resource
}

func (r *Req) GetAct() string {
	return r.Act
}
