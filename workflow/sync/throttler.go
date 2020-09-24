package sync

import (
	"container/heap"
	"sync"
	"time"
)

// Throttler allows CRD controller to limit number of items it is processing in parallel.
type Throttler interface {
	Add(key string, priority int32, creationTime time.Time)
	// Next returns true if item should be processed by controller now or return false.
	Next(key string) bool
	// Remove notifies throttler that item processing is done. In responses the throttler triggers processing of previously throttled items.
	Remove(key string)
}

type throttler struct {
	queue       func(key string)
	inProgress  map[string]bool
	pending     *priorityQueue
	lock        *sync.Mutex
	parallelism int
}

func NewThrottler(parallelism int, queue func(key string)) Throttler {
	return &throttler{
		queue:       queue,
		inProgress:  make(map[string]bool),
		lock:        &sync.Mutex{},
		parallelism: parallelism,
		pending:     &priorityQueue{itemByKey: make(map[string]*item)},
	}
}
func (t *throttler) Add(key string, priority int32, creationTime time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.pending.add(key, priority, creationTime)
	t.queueThrottled()
}

func (t *throttler) Next(key string) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism == 0 || t.inProgress[key] {
		return true
	}
	t.queueThrottled()
	return false
}

func (t *throttler) Remove(key string) {
	t.lock.Lock()
	defer t.lock.Unlock()
	delete(t.inProgress, key)
	t.pending.remove(key)
	t.queueThrottled()
}

func (t *throttler) queueThrottled() {
	for t.parallelism > 0 && t.pending.Len() > 0 && t.parallelism > len(t.inProgress) {
		next := t.pending.pop()
		t.inProgress[next.key] = true
		t.queue(next.key)
	}
}

type item struct {
	key          string
	creationTime time.Time
	priority     int32
	index        int
}

type priorityQueue struct {
	items     []*item
	itemByKey map[string]*item
}

func (pq *priorityQueue) pop() *item {
	return heap.Pop(pq).(*item)
}

func (pq *priorityQueue) peek() *item {
	return pq.items[0]
}

func (pq *priorityQueue) add(key string, priority int32, creationTime time.Time) {
	if res, ok := pq.itemByKey[key]; ok {
		if res.priority != priority {
			res.priority = priority
			heap.Fix(pq, res.index)
		}
	} else {
		heap.Push(pq, &item{key: key, priority: priority, creationTime: creationTime})
	}
}

func (pq *priorityQueue) remove(key string) {
	if item, ok := pq.itemByKey[key]; ok {
		heap.Remove(pq, item.index)
		delete(pq.itemByKey, key)
	}
}

func (pq priorityQueue) Len() int { return len(pq.items) }

func (pq priorityQueue) Less(i, j int) bool {
	if pq.items[i].priority == pq.items[j].priority {
		return pq.items[i].creationTime.Before(pq.items[j].creationTime)
	}
	return pq.items[i].priority > pq.items[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*item)
	item.index = n
	pq.items = append(pq.items, item)
	pq.itemByKey[item.key] = item
}

func (pq *priorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	item.index = -1
	pq.items = old[0 : n-1]
	delete(pq.itemByKey, item.key)
	return item
}
