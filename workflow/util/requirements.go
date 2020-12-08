package util

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func ClusterNameRequirement(clusterName wfv1.ClusterName) labels.Requirement {
	if clusterName != wfv1.ThisCluster {
		r, _ := labels.NewRequirement(common.LabelKeyClusterName, selection.Equals, []string{clusterName})
		return *r
	} else {
		r, _ := labels.NewRequirement(common.LabelKeyClusterName, selection.DoesNotExist, nil)
		return *r
	}
}

func WorkflowNameRequirement(workflowName string) labels.Requirement {
	r, _ := labels.NewRequirement(common.LabelKeyWorkflow, selection.Equals, []string{workflowName})
	return *r
}
