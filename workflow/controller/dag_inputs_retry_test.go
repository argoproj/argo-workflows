package controller

import (
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A reopened native DAG consumes freshly processed arguments, but must not
// rewrite the input evidence of its already completed children.
func TestReopenedDAGRecordsCurrentInputsAndPreservesCompletedChild(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: retry-inputs
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: value
      value: old
  templates:
  - name: main
    inputs:
      parameters:
      - name: value
    dag:
      tasks:
      - name: keep
        template: approve
        arguments:
          parameters:
          - name: value
            value: retained
      - name: wait
        dependencies: [keep]
        template: approve
        arguments:
          parameters:
          - name: value
            value: '{{inputs.parameters.value}}'
  - name: approve
    inputs:
      parameters:
      - name: value
    suspend: {}
`)
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx, *wf)
	woc.operate(ctx)
	root, err := woc.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	require.Equal(t, "old", root.Inputs.Parameters[0].Value.String())
	keep, err := woc.wf.GetNodeByName(wf.Name + ".keep")
	require.NoError(t, err)
	keep.Phase = wfv1.NodeSucceeded
	woc.wf.Status.Nodes[keep.ID] = *keep
	value := wfv1.AnyString("new")
	next := woc.wf.DeepCopy()
	next.Spec.Arguments.Parameters[0].Value = &value
	updated := newWoc(ctx, *next)
	updated.operate(ctx)
	root, err = updated.wf.GetNodeByName(wf.Name)
	require.NoError(t, err)
	require.Equal(t, "new", root.Inputs.Parameters[0].Value.String())
	kept, err := updated.wf.GetNodeByName(wf.Name + ".keep")
	require.NoError(t, err)
	require.Equal(t, keep.Inputs, kept.Inputs)
	require.Equal(t, wfv1.NodeSucceeded, kept.Phase)
	wait, err := updated.wf.GetNodeByName(wf.Name + ".wait")
	require.NoError(t, err)
	require.Equal(t, "new", wait.Inputs.Parameters[0].Value.String())
}
