package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestNodeRequiresHttpReconciliation(t *testing.T) {
	woc := &wfOperationCtx{
		wf: &v1alpha1.Workflow{
			ObjectMeta: v1.ObjectMeta{
				Name: "test-wf",
			},
			Status: v1alpha1.WorkflowStatus{
				Nodes: v1alpha1.Nodes{
					"test-wf-1996333140": v1alpha1.NodeStatus{
						Name: "not-needed",
						Type: v1alpha1.NodeTypePod,
					},
					"test-wf-3939368189": v1alpha1.NodeStatus{
						Name:     "parent",
						Type:     v1alpha1.NodeTypeSteps,
						Children: []string{"test-wf-1430055856"},
					},
					"test-wf-1430055856": v1alpha1.NodeStatus{
						Name: "child-http",
						Type: v1alpha1.NodeTypeHTTP,
					},
				},
			},
		},
	}

	assert.False(t, woc.nodeRequiresTaskSetReconciliation("not-needed"))
	assert.True(t, woc.nodeRequiresTaskSetReconciliation("child-http"))
	assert.True(t, woc.nodeRequiresTaskSetReconciliation("parent"))
}
