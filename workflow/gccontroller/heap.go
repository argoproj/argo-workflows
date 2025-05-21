package gccontroller

import (
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type gcHeap struct {
	heap  []*unstructured.Unstructured
	dedup map[string]bool
}

func NewHeap() *gcHeap {
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

func (h *gcHeap) Push(x interface{}) {
	if _, ok := h.dedup[x.(*unstructured.Unstructured).GetName()]; ok {
		return
	}
	h.dedup[x.(*unstructured.Unstructured).GetName()] = true
	h.heap = append(h.heap, x.(*unstructured.Unstructured))
}

func (h *gcHeap) Pop() interface{} {
	old := h.heap
	n := len(old)
	x := old[n-1]
	h.heap = old[0 : n-1]
	delete(h.dedup, x.GetName())
	return x
}

func (h *gcHeap) PeekPopTimestamp() (time.Time, error) {
	n := len(h.heap)
	if n == 0 {
		return time.Time{}, fmt.Errorf("heap is empty")
	}
	return h.heap[0].GetCreationTimestamp().Time, nil
}
