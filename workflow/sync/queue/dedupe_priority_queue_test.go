package queue

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_dedupePriorityQueue(t *testing.T) {

	q := NewDedupePriorityQueue()

	heap.Push(q, &Item{Value: &val{"f", 1}})
	heap.Push(q, &Item{Value: &val{"e", 2}})
	heap.Push(q, &Item{Value: &val{"d", 3}})
	heap.Push(q, &Item{Value: &val{"c", 4}})
	heap.Push(q, &Item{Value: &val{"b", 5}})
	heap.Push(q, &Item{Value: &val{"a", 6}})

	heap.Init(q)

	assert.Equal(t, 6, q.Len())

	assert.True(t, q.Contains("e"))

	assert.Equal(t, "a", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "b", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "c", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "d", heap.Pop(q).(*Item).Value.(*val).key)

	heap.Remove(q, q.Index("e"))
	assert.Equal(t, "f", heap.Pop(q).(*Item).Value.(*val).key)
}
