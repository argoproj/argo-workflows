package types

import "github.com/argoproj/argo-workflows/v3/workflow/common"

type ClusterRequest interface {
	GetCluster() string
}

func Cluster(req interface{}) string {
	if v, ok := req.(ClusterRequest); ok {
		return v.GetCluster()
	}
	return common.PrimaryCluster()
}
