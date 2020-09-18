package queue

import (
	"container/heap"
)

type Prioritizable interface {
	HigherPriorityThan(x interface{}) bool
}

// Mostly a copy-and-paste from container/heap/example_pq_test.go, but generalized so that any value
// that implements Prioritizable can be used.
//
// While methods receive or return `interface{}`, they always actually required `*Item`.
type PriorityQueue []*Item

var _ heap.Interface = &PriorityQueue{}

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Value.HigherPriorityThan(pq[j].Value)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	item.index = -1
	*pq = old[0 : n-1]
	return item
}

func Peek(h heap.Interface) interface{} {
	x := heap.Pop(h)
	heap.Push(h, x)
	return x
}
