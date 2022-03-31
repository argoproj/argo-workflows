package types

import "github.com/argoproj/argo-workflows/v3/workflow/common"

type Clusterer interface {
	GetCluster() string
}

func Cluster(m interface{}) string {
	if v, ok := m.(Clusterer); ok {
		return v.GetCluster()
	}
	return common.PrimaryCluster()
}
