package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/test/util"
)

func TestUpdater(t *testing.T) {
	wf := &wfv1.Workflow{}
	util.MustUnmarshallYAML(`
status:
  nodes:
    my-wf:
      phase: Succeeded
      children: [my-wf-pod-1, my-wf-pod-2] 
    my-wf-pod-1: 
      phase: Succeeded
      resourcesDuration: 
        my-resource: 1
    my-wf-pod-2: 
      phase: Succeeded
      resourcesDuration: 
        my-resource: 2
`, wf)
	u := NewUpdater(wf)
	u.Init()
	u.Visit("my-wf-pod-1")
	u.Visit("my-wf-pod-2")
	u.Visit("my-wf")
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 1}, wf.Status.Nodes["my-wf-pod-1"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 2}, wf.Status.Nodes["my-wf-pod-2"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 3}, wf.Status.Nodes["my-wf"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"my-resource": 3}, wf.Status.ResourcesDuration)
}
