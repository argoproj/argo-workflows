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
	// the 64-bit hashes are independent of the 32-bit collision
	assert.NotEqual(t, wf.NodeID64(collidingOuterSG), wf.NodeID64(collidingLeafName))
}

func newCollidingWorkflow() *Workflow {
	wf := &Workflow{}
	wf.Name = collidingWfName
	wf.Status.Nodes = Nodes{}
	return wf
}

func (w *Workflow) plant(name string, wide bool) *NodeStatus {
	n := NodeStatus{Name: name}
	if wide {
		n.ID = w.NodeID64(name)
	} else {
		n.ID = w.NodeID(name)
	}
	w.Status.Nodes[n.ID] = n
	return &n
}

func TestResolveNode(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		wf := newCollidingWorkflow()
		node, id := wf.ResolveNode(collidingLeafName)
		assert.Nil(t, node)
		assert.Equal(t, wf.NodeID(collidingLeafName), id)
		assert.Equal(t, wf.NodeID(collidingLeafName), wf.ResolveNodeID(collidingLeafName))
	})

	t.Run("base slot hit", func(t *testing.T) {
		wf := newCollidingWorkflow()
		want := wf.plant(collidingOuterSG, false)
		node, id := wf.ResolveNode(collidingOuterSG)
		require.NotNil(t, node)
		assert.Equal(t, want.ID, node.ID)
		assert.Equal(t, want.ID, id)
	})

	t.Run("base slot held by another name", func(t *testing.T) {
		wf := newCollidingWorkflow()
		winner := wf.plant(collidingOuterSG, false)

		node, id := wf.ResolveNode(collidingLeafName)
		assert.Nil(t, node, "the winner's node must not be returned for the loser's name")
		assert.Equal(t, wf.NodeID64(collidingLeafName), id, "the loser must be created with the widened ID")
		assert.NotEqual(t, winner.ID, id)

		loser := wf.plant(collidingLeafName, true)
		node, id = wf.ResolveNode(collidingLeafName)
		require.NotNil(t, node)
		assert.Equal(t, loser.ID, node.ID)
		assert.Equal(t, loser.ID, id)
		assert.Equal(t, loser.ID, wf.ResolveNodeID(collidingLeafName))

		// the winner is still found under its own name
		node, _ = wf.ResolveNode(collidingOuterSG)
		require.NotNil(t, node)
		assert.Equal(t, winner.ID, node.ID)
	})

	t.Run("widened node found when the base slot is empty", func(t *testing.T) {
		// A retry can delete the node holding the 32-bit slot while the
		// widened node survives; it must still be found.
		wf := newCollidingWorkflow()
		loser := wf.plant(collidingLeafName, true)
		node, id := wf.ResolveNode(collidingLeafName)
		require.NotNil(t, node)
		assert.Equal(t, loser.ID, node.ID)
		assert.Equal(t, loser.ID, id)
	})

	t.Run("both slots held by other names", func(t *testing.T) {
		// A 64-bit collision on top of a 32-bit collision is not resolved:
		// ResolveNode reports no node, and initializeNode refuses to create
		// one because the returned slot is occupied.
		wf := newCollidingWorkflow()
		name := "custom-job-thbh7[0].third"
		wf.Status.Nodes[wf.NodeID(name)] = NodeStatus{Name: collidingOuterSG}
		wf.Status.Nodes[wf.NodeID64(name)] = NodeStatus{Name: collidingLeafName}
		node, id := wf.ResolveNode(name)
		assert.Nil(t, node)
		assert.Equal(t, wf.NodeID64(name), id)
		assert.True(t, wf.Status.Nodes.Has(id), "creation must be refused")
	})
}

func TestGetNodeByNameCollision(t *testing.T) {
	wf := newCollidingWorkflow()
	winner := wf.plant(collidingOuterSG, false)

	_, err := wf.GetNodeByName(collidingLeafName)
	require.Error(t, err, "must not return the colliding node")

	loser := wf.plant(collidingLeafName, true)
	node, err := wf.GetNodeByName(collidingLeafName)
	require.NoError(t, err)
	assert.Equal(t, loser.ID, node.ID)
	assert.Equal(t, collidingLeafName, node.Name)
	node, err = wf.GetNodeByName(collidingOuterSG)
	require.NoError(t, err)
	assert.Equal(t, winner.ID, node.ID)
}

// BenchmarkResolveNodeMiss measures the not-found path, which every node pays
// once before it is created: the 32-bit slot plus the 64-bit fallback.
func BenchmarkResolveNodeMiss(b *testing.B) {
	wf := newCollidingWorkflow()
	for i := range 10000 {
		wf.plant("custom-job-thbh7[0].fanout("+strconv.Itoa(i)+":item)", false)
	}
	name := "custom-job-thbh7[0].fanout(99999:item)"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if n, _ := wf.ResolveNode(name); n != nil {
			b.Fatal("unexpected hit")
		}
	}
}
