package types

import "github.com/argoproj/argo-workflows/v3/workflow/common"

type Clusterer interface {
	GetCluster() string
}

func Cluster(req interface{}) string {
	if v, ok := req.(Clusterer); ok {
		return v.GetCluster()
	}
	return common.PrimaryCluster()
}
