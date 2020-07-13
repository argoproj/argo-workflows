package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniqueQueue(t *testing.T) {
	queue := newUniquePhaseNodeQueue()
	assert.True(t, queue.empty())

	phaseNodeA := phaseNode{nodeId: "node-a"}
	queue.add(phaseNodeA)
	assert.Equal(t, 1, queue.len())
	assert.False(t, queue.empty())
	queue.add(phaseNodeA)
	assert.Equal(t, 1, queue.len())

	phaseNodeB := phaseNode{nodeId: "node-b"}
	queue.add(phaseNodeB)
	assert.Equal(t, 2, queue.len())
	queue.add(phaseNodeB)
	assert.Equal(t, 2, queue.len())

	pop := queue.pop()
	assert.Equal(t, "node-a", pop.nodeId)
	assert.Equal(t, 1, queue.len())
	pop = queue.pop()
	assert.True(t, queue.empty())
	assert.Equal(t, "node-b", pop.nodeId)
	assert.Equal(t, 0, queue.len())

	queue.add(phaseNodeA)
	assert.Equal(t, 0, queue.len())
	queue.add(phaseNodeB)
	assert.Equal(t, 0, queue.len())
}

func TestUniqueQueueConstructor(t *testing.T) {
	phaseNodeA := phaseNode{nodeId: "node-a"}
	queue := newUniquePhaseNodeQueue(phaseNodeA)
	assert.Equal(t, 1, queue.len())
	assert.False(t, queue.empty())
	queue.add(phaseNodeA)
	assert.Equal(t, 1, queue.len())

	phaseNodeB := phaseNode{nodeId: "node-b"}
	queue.add(phaseNodeB)
	assert.Equal(t, 2, queue.len())
	queue.add(phaseNodeB)
	assert.Equal(t, 2, queue.len())

	pop := queue.pop()
	assert.Equal(t, "node-a", pop.nodeId)
	assert.Equal(t, 1, queue.len())
	pop = queue.pop()
	assert.True(t, queue.empty())
	assert.Equal(t, "node-b", pop.nodeId)
	assert.Equal(t, 0, queue.len())

	queue.add(phaseNodeA)
	assert.Equal(t, 0, queue.len())
	queue.add(phaseNodeB)
	assert.Equal(t, 0, queue.len())
}
