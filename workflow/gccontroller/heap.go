package gccontroller

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Heap is the interface for a GC heap (implements heap.Interface)
type Heap interface {
	Len() int
	Less(i, j int) bool
	Swap(i, j int)
	Push(x any)
	Pop() any
}

type gcHeap struct {
	heap  []*unstructured.Unstructured
	dedup map[string]bool
}

func NewHeap() Heap {
	return &gcHeap{
		heap:  make([]*unstructured.Unstructured, 0),
		dedup: make(map[string]bool),
	}
}

func (h *gcHeap) Len() int { return len(h.heap) }
func (h *gcHeap) Less(i, j int) bool {
	return h.heap[j].GetCreationTimestamp().After((h.heap[i].GetCreationTimestamp().Time))
}
func (h *gcHeap) Swap(i, j int) { h.heap[i], h.heap[j] = h.heap[j], h.heap[i] }

func (h *gcHeap) Push(x any) {
	if _, ok := h.dedup[x.(*unstructured.Unstructured).GetName()]; ok {
		return
	}
	h.dedup[x.(*unstructured.Unstructured).GetName()] = true
	h.heap = append(h.heap, x.(*unstructured.Unstructured))
}

func (h *gcHeap) Pop() any {
	old := h.heap
	n := len(old)
	x := old[n-1]
	h.heap = old[0 : n-1]
	delete(h.dedup, x.GetName())
	return x
}
