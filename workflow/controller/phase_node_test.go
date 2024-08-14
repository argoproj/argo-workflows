package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUniqueQueue(t *testing.T) {
	queue := newUniquePhaseNodeQueue()
	require.True(t, queue.empty())

	phaseNodeA := phaseNode{nodeId: "node-a"}
	queue.add(phaseNodeA)
	require.Equal(t, 1, queue.len())
	require.False(t, queue.empty())
	queue.add(phaseNodeA)
	require.Equal(t, 1, queue.len())

	phaseNodeB := phaseNode{nodeId: "node-b"}
	queue.add(phaseNodeB)
	require.Equal(t, 2, queue.len())
	queue.add(phaseNodeB)
	require.Equal(t, 2, queue.len())

	pop := queue.pop()
	require.Equal(t, "node-a", pop.nodeId)
	require.Equal(t, 1, queue.len())
	pop = queue.pop()
	require.True(t, queue.empty())
	require.Equal(t, "node-b", pop.nodeId)
	require.Equal(t, 0, queue.len())

	queue.add(phaseNodeA)
	require.Equal(t, 0, queue.len())
	queue.add(phaseNodeB)
	require.Equal(t, 0, queue.len())
}

func TestUniqueQueueConstructor(t *testing.T) {
	phaseNodeA := phaseNode{nodeId: "node-a"}
	queue := newUniquePhaseNodeQueue(phaseNodeA)
	require.Equal(t, 1, queue.len())
	require.False(t, queue.empty())
	queue.add(phaseNodeA)
	require.Equal(t, 1, queue.len())

	phaseNodeB := phaseNode{nodeId: "node-b"}
	queue.add(phaseNodeB)
	require.Equal(t, 2, queue.len())
	queue.add(phaseNodeB)
	require.Equal(t, 2, queue.len())

	pop := queue.pop()
	require.Equal(t, "node-a", pop.nodeId)
	require.Equal(t, 1, queue.len())
	pop = queue.pop()
	require.True(t, queue.empty())
	require.Equal(t, "node-b", pop.nodeId)
	require.Equal(t, 0, queue.len())

	queue.add(phaseNodeA)
	require.Equal(t, 0, queue.len())
	queue.add(phaseNodeB)
	require.Equal(t, 0, queue.len())
}
