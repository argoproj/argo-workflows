package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A workflow whose node names collide, see
// https://github.com/argoproj/argo-workflows/issues/16376. The leaf lost the
// collision and carries HashSuffix 1.
func collidingWorkflow(t *testing.T) *wfv1.Workflow {
	t.Helper()
	wf := &wfv1.Workflow{}
	wf.Name = "custom-job-thbh7"
	wf.Status.Phase = wfv1.WorkflowFailed
	wf.Status.Nodes = wfv1.Nodes{}
	plant := func(name string, suffix int32, typ wfv1.NodeType, children ...string) *wfv1.NodeStatus {
		n := wfv1.NodeStatus{Name: name, HashSuffix: suffix, Type: typ, Phase: wfv1.NodeFailed, Children: children}
		n.ID = wf.NodeID(n.HashName())
		wf.Status.Nodes[n.ID] = n
		return &n
	}
	leaf := plant("custom-job-thbh7[0].custom-job[0].custom-job-main", 1, wfv1.NodeTypePod)
	innerSG := plant("custom-job-thbh7[0].custom-job[0]", 0, wfv1.NodeTypeStepGroup, leaf.ID)
	steps := plant("custom-job-thbh7[0].custom-job", 0, wfv1.NodeTypeSteps, innerSG.ID)
	outerSG := plant("custom-job-thbh7[0]", 0, wfv1.NodeTypeStepGroup, steps.ID)
	plant("custom-job-thbh7", 0, wfv1.NodeTypeSteps, outerSG.ID)
	require.Equal(t, wf.NodeID(outerSG.Name), wf.NodeID(leaf.Name), "names must collide")
	return wf
}

func TestFormulateResubmitWorkflowKeepsHashSuffix(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := collidingWorkflow(t)

	newWf, err := FormulateResubmitWorkflow(ctx, wf, true, nil)
	require.NoError(t, err)
	require.NotEqual(t, wf.Name, newWf.Name)
	assert.Len(t, newWf.Status.Nodes, len(wf.Status.Nodes))

	leafName := newWf.Name + "[0].custom-job[0].custom-job-main"
	leaf, err := newWf.GetNodeByName(leafName)
	require.NoError(t, err)
	assert.Equal(t, int32(1), leaf.HashSuffix)
	assert.Equal(t, newWf.NodeID(leafName+"~1"), leaf.ID)

	innerSG, err := newWf.GetNodeByName(newWf.Name + "[0].custom-job[0]")
	require.NoError(t, err)
	assert.Equal(t, []string{leaf.ID}, innerSG.Children, "child references must be converted with the suffix")

	for _, n := range newWf.Status.Nodes {
		assert.Equal(t, newWf.NodeID(n.HashName()), n.ID)
	}
}

func TestFormulateRetryWorkflowKeepsHashSuffix(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := collidingWorkflow(t)
	leafName := "custom-job-thbh7[0].custom-job[0].custom-job-main"

	newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
	require.NoError(t, err)

	// retry drops the failed pod node so it is re-run; the step group that
	// won the base slot is kept, so the leaf must resolve to the suffixed slot
	// again when it is recreated
	_, err = newWf.GetNodeByName("custom-job-thbh7[0]")
	require.NoError(t, err)
	leaf, suffix := newWf.ResolveNode(leafName)
	assert.Nil(t, leaf)
	assert.Equal(t, int32(1), suffix)
	for _, n := range newWf.Status.Nodes {
		assert.Equal(t, newWf.NodeID(n.HashName()), n.ID)
	}
}
