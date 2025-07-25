package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestUpdater(t *testing.T) {
	wf := &wfv1.Workflow{}
	ctx := logging.TestContext(t.Context())
	wfv1.MustUnmarshal(`
status:
  nodes:
    root:
      phase: Succeeded
      children: [pod, dag] 
    pod: 
      phase: Succeeded
      type: Pod
      resourcesDuration: 
        x: 1
      children: [dag]
    dag: 
      phase: Succeeded
      children: [dag-pod]
    dag-pod: 
      phase: Succeeded
      type: Pod
      resourcesDuration: 
        x: 2
`, wf)
	UpdateResourceDurations(ctx, wf)
	assert.Equal(t, wfv1.ResourcesDuration{"x": 2}, wf.Status.Nodes["dag-pod"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"x": 2}, wf.Status.Nodes["dag"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"x": 1}, wf.Status.Nodes["pod"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"x": 3}, wf.Status.Nodes["root"].ResourcesDuration)
	assert.Equal(t, wfv1.ResourcesDuration{"x": 3}, wf.Status.ResourcesDuration)
}
