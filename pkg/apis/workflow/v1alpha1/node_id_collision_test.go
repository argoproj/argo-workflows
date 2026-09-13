package v1alpha1

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Two real node names from a workflow named custom-job-thbh7 whose FNV-32a
// hashes are equal, from https://github.com/argoproj/argo-workflows/issues/16376
const (
	collidingWfName   = "custom-job-thbh7"
	collidingOuterSG  = "custom-job-thbh7[0]"
	collidingLeafName = "custom-job-thbh7[0].custom-job[0].custom-job-main"
)

func TestNodeIDCollisionPair(t *testing.T) {
	wf := &Workflow{}
	wf.Name = collidingWfName
	assert.Equal(t, wf.NodeID(collidingOuterSG), wf.NodeID(collidingLeafName), "the pair this file relies on must collide")
	// FNV-1a has no finalisation, so a collision extends to every common suffix
	assert.Equal(t, wf.NodeID(collidingOuterSG+"[0]"), wf.NodeID(collidingLeafName+"[0]"))
	assert.NotEqual(t, wf.NodeID(collidingOuterSG), wf.NodeID(collidingLeafName+"~1"))
}

func TestHashName(t *testing.T) {
	assert.Equal(t, "a", NodeStatus{Name: "a"}.HashName())
	assert.Equal(t, "a~1", NodeStatus{Name: "a", HashSuffix: 1}.HashName())
	assert.Equal(t, "a~12", NodeStatus{Name: "a", HashSuffix: 12}.HashName())
}

func newCollidingWorkflow() *Workflow {
	wf := &Workflow{}
	wf.Name = collidingWfName
	wf.Status.Nodes = Nodes{}
	return wf
}

func (w *Workflow) plant(name string, suffix int32) *NodeStatus {
	n := NodeStatus{Name: name, HashSuffix: suffix}
	n.ID = w.NodeID(n.HashName())
	w.Status.Nodes[n.ID] = n
	return &n
}

func TestResolveNode(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		wf := newCollidingWorkflow()
		node, suffix := wf.ResolveNode(collidingLeafName)
		assert.Nil(t, node)
		assert.Equal(t, int32(0), suffix)
		assert.Equal(t, wf.NodeID(collidingLeafName), wf.ResolveNodeID(collidingLeafName))
	})

	t.Run("base slot hit", func(t *testing.T) {
		wf := newCollidingWorkflow()
		want := wf.plant(collidingOuterSG, 0)
		node, suffix := wf.ResolveNode(collidingOuterSG)
		require.NotNil(t, node)
		assert.Equal(t, want.ID, node.ID)
		assert.Equal(t, int32(0), suffix)
	})

	t.Run("base slot held by another name", func(t *testing.T) {
		wf := newCollidingWorkflow()
		winner := wf.plant(collidingOuterSG, 0)

		node, suffix := wf.ResolveNode(collidingLeafName)
		assert.Nil(t, node, "the winner's node must not be returned for the loser's name")
		assert.Equal(t, int32(1), suffix)
		loserID := wf.ResolveNodeID(collidingLeafName)
		assert.NotEqual(t, winner.ID, loserID)
		assert.Equal(t, wf.NodeID(collidingLeafName+"~1"), loserID)

		loser := wf.plant(collidingLeafName, 1)
		assert.Equal(t, loserID, loser.ID)
		node, suffix = wf.ResolveNode(collidingLeafName)
		require.NotNil(t, node)
		assert.Equal(t, loser.ID, node.ID)
		assert.Equal(t, int32(1), suffix)
		assert.Equal(t, loser.ID, wf.ResolveNodeID(collidingLeafName))

		// the winner is still found under its own name
		node, _ = wf.ResolveNode(collidingOuterSG)
		require.NotNil(t, node)
		assert.Equal(t, winner.ID, node.ID)
	})

	t.Run("peek when the base slot has been emptied", func(t *testing.T) {
		// A retry can delete the node holding the base slot while the suffixed
		// node survives, and a resubmit renames everything so the two names
		// no longer collide. Either way the suffixed node must still be found.
		wf := newCollidingWorkflow()
		loser := wf.plant(collidingLeafName, 1)
		node, suffix := wf.ResolveNode(collidingLeafName)
		require.NotNil(t, node)
		assert.Equal(t, loser.ID, node.ID)
		assert.Equal(t, int32(1), suffix)
		assert.Equal(t, loser.ID, wf.ResolveNodeID(collidingLeafName))
	})

	t.Run("family of three", func(t *testing.T) {
		// Plant nodes belonging to other names on the first two slots of
		// name's chain. ResolveNode only looks at what is stored at each
		// slot, so the planted nodes need not hash there themselves.
		wf := newCollidingWorkflow()
		name := "custom-job-thbh7[0].third"
		wf.Status.Nodes[wf.NodeID(name)] = NodeStatus{Name: collidingOuterSG}
		wf.Status.Nodes[wf.NodeID(name+"~1")] = NodeStatus{Name: collidingLeafName, HashSuffix: 1}
		node, suffix := wf.ResolveNode(name)
		assert.Nil(t, node)
		assert.Equal(t, int32(2), suffix)
		third := wf.plant(name, 2)
		node, suffix = wf.ResolveNode(name)
		require.NotNil(t, node)
		assert.Equal(t, third.ID, node.ID)
		assert.Equal(t, int32(2), suffix)
	})

	t.Run("walk past a hole followed by a foreign slot", func(t *testing.T) {
		// A single deletion can empty a slot while a different name still
		// holds the next one; the survivor behind them must still be found.
		wf := newCollidingWorkflow()
		name := "custom-job-thbh7[0].third"
		wf.Status.Nodes[wf.NodeID(name+"~1")] = NodeStatus{Name: collidingLeafName, HashSuffix: 1}
		third := wf.plant(name, 2)
		node, suffix := wf.ResolveNode(name)
		require.NotNil(t, node)
		assert.Equal(t, third.ID, node.ID)
		assert.Equal(t, int32(2), suffix)
	})

	t.Run("a hole is reused for allocation", func(t *testing.T) {
		wf := newCollidingWorkflow()
		name := "custom-job-thbh7[0].third"
		wf.Status.Nodes[wf.NodeID(name+"~1")] = NodeStatus{Name: collidingLeafName, HashSuffix: 1}
		node, suffix := wf.ResolveNode(name)
		assert.Nil(t, node)
		assert.Equal(t, int32(0), suffix)
	})

	t.Run("matches on name and suffix separately", func(t *testing.T) {
		// Hook names are not charset validated, so a spec-derived name can
		// itself end in "~1". It must never satisfy a lookup for the name
		// without the suffix.
		wf := newCollidingWorkflow()
		hookX := "custom-job-thbh7.hooks.x"
		wf.Status.Nodes[wf.NodeID(hookX)] = NodeStatus{Name: collidingOuterSG}
		hookX1 := wf.plant(hookX+"~1", 0)

		node, suffix := wf.ResolveNode(hookX)
		assert.Nil(t, node)
		assert.Equal(t, int32(2), suffix)

		node, suffix = wf.ResolveNode(hookX + "~1")
		require.NotNil(t, node)
		assert.Equal(t, hookX1.ID, node.ID)
		assert.Equal(t, int32(0), suffix)
	})
}

func TestGetNodeByNameCollision(t *testing.T) {
	wf := newCollidingWorkflow()
	winner := wf.plant(collidingOuterSG, 0)

	_, err := wf.GetNodeByName(collidingLeafName)
	require.Error(t, err, "must not return the colliding node")

	loser := wf.plant(collidingLeafName, 1)
	node, err := wf.GetNodeByName(collidingLeafName)
	require.NoError(t, err)
	assert.Equal(t, loser.ID, node.ID)
	assert.Equal(t, collidingLeafName, node.Name)
	node, err = wf.GetNodeByName(collidingOuterSG)
	require.NoError(t, err)
	assert.Equal(t, winner.ID, node.ID)
}

// BenchmarkResolveNodeMiss measures the not-found path, which every node pays
// once before it is created: base slot lookup plus the one-slot peek.
func BenchmarkResolveNodeMiss(b *testing.B) {
	wf := newCollidingWorkflow()
	for i := range 10000 {
		wf.plant("custom-job-thbh7[0].fanout("+strconv.Itoa(i)+":item)", 0)
	}
	name := "custom-job-thbh7[0].fanout(99999:item)"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if n, _ := wf.ResolveNode(name); n != nil {
			b.Fatal("unexpected hit")
		}
	}
}
