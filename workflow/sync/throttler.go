package sync

import (
	"container/heap"
	"sync"

	"github.com/argoproj/argo/workflow/sync/queue"
)

// Throttler allows CRD controller to limit number of Items it is processing in parallel.
type Throttler interface {
	Add(x queue.Keyed)
	// Accept returns true if Item should be processed.
	Next(key string) bool
	// Remove notifies throttler that Item processing is done. The throttler will re-queue the next Item.
	Remove(key string)
}

type throttler struct {
	requeue     func(key string)
	inProgress  map[string]bool
	pending     *queue.DedupePriorityQueue
	lock        *sync.Mutex
	parallelism int
}

func NewThrottler(parallelism int, requeue func(key string)) Throttler {
	return &throttler{
		requeue:     requeue,
		inProgress:  make(map[string]bool),
		lock:        &sync.Mutex{},
		parallelism: parallelism,
		pending:     queue.NewDedupePriorityQueue(),
	}
}

func (t *throttler) Add(x queue.Keyed) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism < 1 {
		return
	}
	if t.pending.Contains(x.GetKey()) {
		heap.Remove(t.pending, t.pending.Index(x.GetKey()))
	}
	heap.Push(t.pending, queue.NewItem(x))
}

func (t *throttler) Next(key string) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism < 1 || t.inProgress[key] {
		return true
	}
	if t.parallelism <= len(t.inProgress) {
		return false
	}
	next := heap.Pop(t.pending).(*queue.Item)
	if next.Value.(queue.Keyed).GetKey() == key {
		t.inProgress[key] = true
		return true
	}
	heap.Push(t.pending, next)
	return false
}

func (t *throttler) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism < 1 {
		return
	}
	delete(t.inProgress, key)
	if t.pending.Contains(key) {
		heap.Remove(t.pending, t.pending.Index(key))
	}
	t.queueThrottled()
}

func (t *throttler) queueThrottled() {
	for t.parallelism > len(t.inProgress) && t.pending.Len() > 0 {
		k := heap.Pop(t.pending).(*queue.Item).Value.(queue.Keyed)
		t.inProgress[k.GetKey()] = true
		t.requeue(k.GetKey())
	}
}
