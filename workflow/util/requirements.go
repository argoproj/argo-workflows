package util

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

func ClusterNameRequirement(a, b wfv1.ClusterName) labels.Requirement {
	if a != b {
		r, _ := labels.NewRequirement(common.LabelKeyWorkflowClusterName, selection.Equals, []string{string(b)})
		return *r
	} else {
		r, _ := labels.NewRequirement(common.LabelKeyWorkflowClusterName, selection.DoesNotExist, nil)
		return *r
	}
}

func WorkflowNamespaceRequirement(namespace string) labels.Requirement {
	r, _ := labels.NewRequirement(common.LabelKeyWorkflowNamespace, selection.Equals, []string{namespace})
	return *r
}

func WorkflowNameRequirement(name string) labels.Requirement {
	r, _ := labels.NewRequirement(common.LabelKeyWorkflow, selection.Equals, []string{name})
	return *r
}
