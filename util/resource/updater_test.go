package resource

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestUpdater(t *testing.T) {
	wf := &wfv1.Workflow{}
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
	UpdateResourceDurations(wf)
	require.Equal(t, wfv1.ResourcesDuration{"x": 2}, wf.Status.Nodes["dag-pod"].ResourcesDuration)
	require.Equal(t, wfv1.ResourcesDuration{"x": 2}, wf.Status.Nodes["dag"].ResourcesDuration)
	require.Equal(t, wfv1.ResourcesDuration{"x": 1}, wf.Status.Nodes["pod"].ResourcesDuration)
	require.Equal(t, wfv1.ResourcesDuration{"x": 3}, wf.Status.Nodes["root"].ResourcesDuration)
	require.Equal(t, wfv1.ResourcesDuration{"x": 3}, wf.Status.ResourcesDuration)
}
