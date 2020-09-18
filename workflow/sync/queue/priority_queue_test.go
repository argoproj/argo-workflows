package queue

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	q := &PriorityQueue{}

	heap.Push(q, &Item{Value: &val{"f", 1}})
	e := &Item{Value: &val{"e", 2}}
	heap.Push(q, e)
	heap.Push(q, &Item{Value: &val{"d", 3}})
	heap.Push(q, &Item{Value: &val{"c", 4}})
	heap.Push(q, &Item{Value: &val{"b", 5}})
	heap.Push(q, &Item{Value: &val{"a", 6}})

	assert.Equal(t, 6, q.Len())

	assert.Equal(t, "a", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "b", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "c", heap.Pop(q).(*Item).Value.(*val).key)
	assert.Equal(t, "d", heap.Pop(q).(*Item).Value.(*val).key)

	heap.Remove(q, e.index)
	assert.Equal(t, "f", heap.Pop(q).(*Item).Value.(*val).key)
}
