package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestUpdator(t *testing.T) {
	wf := &wfv1.Workflow{
		Status: wfv1.WorkflowStatus{
			Nodes: wfv1.Nodes{
				"my-wf-pod-1": wfv1.NodeStatus{Type: wfv1.NodeTypePod, ResourcesDuration: wfv1.ResourcesDuration{"my-resource": 1}},
				"my-wf-pod-2": wfv1.NodeStatus{Type: wfv1.NodeTypePod, ResourcesDuration: wfv1.ResourcesDuration{"my-resource": 1}},
				"my-wf":       wfv1.NodeStatus{Phase: wfv1.NodeSucceeded, Type: wfv1.NodeTypeDAG, Children: []string{"my-wf-pod-1", "my-wf-pod-2"}},
			},
		},
	}
	u := NewUpdator(wf)
	u.Init()
	u.Visit("my-wf-pod-1")
	u.Visit("my-wf-pod-2")
	u.Visit("my-wf")
	assert.True(t, u.Updated)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 1}, wf.Status.Nodes["my-wf-pod-1"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 1}, wf.Status.Nodes["my-wf-pod-2"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 2}, wf.Status.Nodes["my-wf"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 2}, wf.Status.ResourcesDuration)
}
