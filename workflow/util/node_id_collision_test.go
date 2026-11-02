package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// A workflow whose node names collide, see
// https://github.com/argoproj/argo-workflows/issues/16376. The leaf lost the
// collision and carries the widened 64-bit ID.
func collidingWorkflow(t *testing.T, phase wfv1.NodePhase) *wfv1.Workflow {
	t.Helper()
	wf := &wfv1.Workflow{}
	wf.Name = "custom-job-thbh7"
	wf.Status.Phase = wfv1.WorkflowFailed
	wf.Status.Nodes = wfv1.Nodes{}
	plant := func(name string, wide bool, typ wfv1.NodeType, children ...string) *wfv1.NodeStatus {
		n := wfv1.NodeStatus{Name: name, Type: typ, Phase: phase, Children: children}
		if wide {
			n.ID = wf.NodeID64(name)
		} else {
			n.ID = wf.NodeID(name)
		}
		wf.Status.Nodes[n.ID] = n
		return &n
	}
	leaf := plant("custom-job-thbh7[0].custom-job[0].custom-job-main", true, wfv1.NodeTypePod)
	innerSG := plant("custom-job-thbh7[0].custom-job[0]", false, wfv1.NodeTypeStepGroup, leaf.ID)
	steps := plant("custom-job-thbh7[0].custom-job", false, wfv1.NodeTypeSteps, innerSG.ID)
	outerSG := plant("custom-job-thbh7[0]", false, wfv1.NodeTypeStepGroup, steps.ID)
	plant("custom-job-thbh7", false, wfv1.NodeTypeSteps, outerSG.ID)
	require.Equal(t, wf.NodeID(outerSG.Name), wf.NodeID(leaf.Name), "names must collide")
	return wf
}

func TestFormulateResubmitWorkflowWithCollision(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	// nothing failed below the root, so memoized resubmit keeps every node and
	// has to carry the widened leaf ID across the rename
	wf := collidingWorkflow(t, wfv1.NodeSucceeded)

	newWf, err := FormulateResubmitWorkflow(ctx, wf, true, nil)
	require.NoError(t, err)
	require.NotEqual(t, wf.Name, newWf.Name)
	assert.Len(t, newWf.Status.Nodes, len(wf.Status.Nodes))

	// the rename changes the hash input, so the old collision dissolves and
	// every node goes back to its 32-bit slot
	leafName := newWf.Name + "[0].custom-job[0].custom-job-main"
	leaf, err := newWf.GetNodeByName(leafName)
	require.NoError(t, err)
	assert.Equal(t, newWf.NodeID(leafName), leaf.ID)

	innerSG, err := newWf.GetNodeByName(newWf.Name + "[0].custom-job[0]")
	require.NoError(t, err)
	assert.Equal(t, []string{leaf.ID}, innerSG.Children, "child references must be converted")

	for _, n := range newWf.Status.Nodes {
		assert.Equal(t, newWf.NodeID(n.Name), n.ID)
	}
}

func TestFormulateResubmitWorkflowWithCollisionFailedLeaf(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := collidingWorkflow(t, wfv1.NodeFailed)

	newWf, err := FormulateResubmitWorkflow(ctx, wf, true, nil)
	require.NoError(t, err)

	// memoized resubmit plans the same reset as a retry, so the failed pod
	// node is dropped for the controller to create afresh, taking the widened
	// ID with it
	_, err = newWf.GetNodeByName(newWf.Name + "[0].custom-job[0].custom-job-main")
	require.Error(t, err)
	innerSG, err := newWf.GetNodeByName(newWf.Name + "[0].custom-job[0]")
	require.NoError(t, err)
	assert.Empty(t, innerSG.Children, "the reference to the deleted node must go")
	for _, n := range newWf.Status.Nodes {
		assert.Equal(t, newWf.NodeID(n.Name), n.ID)
	}
}

func TestFormulateRetryWorkflowWithCollision(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	wf := collidingWorkflow(t, wfv1.NodeFailed)
	leafName := "custom-job-thbh7[0].custom-job[0].custom-job-main"

	newWf, _, err := FormulateRetryWorkflow(ctx, wf, false, "", nil)
	require.NoError(t, err)

	// retry drops the failed pod node so it is re-run; the step group that
	// won the 32-bit slot is kept, so the leaf must resolve to the widened
	// slot again when it is recreated
	_, err = newWf.GetNodeByName("custom-job-thbh7[0]")
	require.NoError(t, err)
	leaf, id := newWf.ResolveNode(leafName)
	assert.Nil(t, leaf)
	assert.Equal(t, newWf.NodeID64(leafName), id)
	for _, n := range newWf.Status.Nodes {
		assert.Contains(t, []string{newWf.NodeID(n.Name), newWf.NodeID64(n.Name)}, n.ID)
	}
}

// Two sibling node names of a withItems fan-out that collide under NodeID
// once the workflow is called custom-job-abcde. Renaming a workflow rehashes
// every node name, so a resubmit can create a collision that the original
// workflow did not have; these were found by searching for one.
const (
	renamedWfName     = "custom-job-abcde"
	collidingFanoutA  = "[0].fanout(26094:item26094)"
	collidingFanoutB  = "[0].fanout(35221:item35221)"
	uncollidingWfName = "custom-job-thbh7"
)

func TestNewNodeIDsWidensACollisionMadeByTheRename(t *testing.T) {
	wf := &wfv1.Workflow{}
	wf.Name = uncollidingWfName
	wf.Status.Nodes = wfv1.Nodes{}
	for _, suffix := range []string{"", collidingFanoutA, collidingFanoutB} {
		name := wf.Name + suffix
		wf.Status.Nodes[wf.NodeID(name)] = wfv1.NodeStatus{Name: name, ID: wf.NodeID(name)}
	}
	require.Len(t, wf.Status.Nodes, 3, "the names must not collide under the old workflow name")

	newWf := &wfv1.Workflow{}
	newWf.Name = renamedWfName
	nameA, nameB := newWf.Name+collidingFanoutA, newWf.Name+collidingFanoutB
	require.Equal(t, newWf.NodeID(nameA), newWf.NodeID(nameB), "the names must collide under the new workflow name")

	rename := func(name string) string { return newWf.Name + strings.TrimPrefix(name, wf.Name) }
	newIDs, err := newNodeIDs(wf, newWf, rename)
	require.NoError(t, err)

	// the first name in name order keeps the 32-bit slot, the loser is widened
	assert.Equal(t, newWf.NodeID(nameA), newIDs[wf.NodeID(wf.Name+collidingFanoutA)])
	assert.Equal(t, newWf.NodeID64(nameB), newIDs[wf.NodeID(wf.Name+collidingFanoutB)])

	// which is where the resubmitted workflow can find them again
	newWf.Status.Nodes = wfv1.Nodes{}
	for oldID, newID := range newIDs {
		newWf.Status.Nodes[newID] = wfv1.NodeStatus{Name: rename(wf.Status.Nodes[oldID].Name), ID: newID}
	}
	assert.Len(t, newWf.Status.Nodes, len(wf.Status.Nodes), "no node may be lost to a collision")
	for _, name := range []string{newWf.Name, nameA, nameB} {
		node, err := newWf.GetNodeByName(name)
		require.NoError(t, err)
		assert.Equal(t, name, node.Name)
	}
}

func TestNewNodeIDsKeepsDeletedNodesOutOfTheWay(t *testing.T) {
	wf := collidingWorkflow(t, wfv1.NodeFailed)
	newWf := &wfv1.Workflow{}
	newWf.Name = renamedWfName
	rename := func(name string) string { return newWf.Name + strings.TrimPrefix(name, wf.Name) }

	newIDs, err := newNodeIDs(wf, newWf, rename)
	require.NoError(t, err)

	// every node is mapped, including ones a reset plan goes on to delete, and
	// no two share an ID, so a dangling reference cannot alias a kept node
	assert.Len(t, newIDs, len(wf.Status.Nodes))
	seen := map[string]bool{}
	for _, id := range newIDs {
		assert.False(t, seen[id], "IDs must be unique")
		seen[id] = true
	}
}
