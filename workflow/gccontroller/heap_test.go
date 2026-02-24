package gccontroller

import (
	"container/heap"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	testutil "github.com/argoproj/argo-workflows/v4/test/util"
)

func TestPriorityQueue(t *testing.T) {
	wf := testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	now := time.Now()
	wf.SetCreationTimestamp(v1.Time{Time: now})
	queue := &gcHeap{
		heap:  []*unstructured.Unstructured{wf},
		dedup: make(map[string]bool),
	}
	heap.Init(queue)
	assert.Equal(t, 1, queue.Len())
	wf = testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	wf.SetCreationTimestamp(v1.Time{Time: now.Add(time.Second)})
	heap.Push(queue, wf)
	wf = testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline-oldest
  labels:
    workflows.argoproj.io/phase: Failed
`)
	wf.SetCreationTimestamp(v1.Time{Time: now.Add(-time.Second)})
	heap.Push(queue, wf)
	assert.Equal(t, 3, queue.Len())
	first := heap.Pop(queue).(*unstructured.Unstructured)
	assert.Equal(t, now.Add(-time.Second).Unix(), first.GetCreationTimestamp().Unix())
	assert.Equal(t, "bad-baseline-oldest", first.GetName())
	assert.Equal(t, now.Unix(), heap.Pop(queue).(*unstructured.Unstructured).GetCreationTimestamp().Unix())
	assert.Equal(t, now.Add(time.Second).Unix(), heap.Pop(queue).(*unstructured.Unstructured).GetCreationTimestamp().Unix())
	assert.Equal(t, 0, queue.Len())
}

func TestDeduplicationOfPriorityQueue(t *testing.T) {
	wf := testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	now := time.Now()
	wf.SetCreationTimestamp(v1.Time{Time: now})
	queue := &gcHeap{
		heap:  []*unstructured.Unstructured{},
		dedup: make(map[string]bool),
	}
	heap.Push(queue, wf)
	assert.Equal(t, 1, queue.Len())
	wf = testutil.MustUnmarshalUnstructured(`
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  name: bad-baseline
  labels:
    workflows.argoproj.io/phase: Failed
`)
	wf.SetCreationTimestamp(v1.Time{Time: now.Add(-time.Second)})
	heap.Push(queue, wf)
	assert.Equal(t, 1, queue.Len())
	_ = heap.Pop(queue)
	assert.Equal(t, 0, queue.Len())
	heap.Push(queue, wf)
	assert.Equal(t, 1, queue.Len())
}
