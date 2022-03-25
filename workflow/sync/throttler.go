package sync

import (
	"container/heap"
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

//go:generate mockery --name=Throttler

// Throttler allows the controller to limit number of items it is processing in parallel.
// Items are processed in priority order, and one processing starts, other items (including higher-priority items)
// will be kept pending until the processing is complete.
// Implementations should be idempotent.
type Throttler interface {
	Init(wfs []wfv1.Workflow) error
	Add(key Key, priority int32, creationTime time.Time)
	// Admit returns if the item should be processed.
	Admit(key Key) bool
	// Remove notifies throttler that item processing is no longer needed
	Remove(key Key)
}

type Key = string
type QueueFunc func(Key)

type BucketKey = string
type BucketFunc func(Key) BucketKey

var SingleBucket BucketFunc = func(key Key) BucketKey { return "" }
var NamespaceBucket BucketFunc = func(key Key) BucketKey {
	namespace, _, _ := cache.SplitMetaNamespaceKey(key)
	return namespace
}

type throttler struct {
	queue       QueueFunc
	bucketFunc  BucketFunc
	inProgress  buckets
	pending     map[BucketKey]*priorityQueue
	lock        *sync.Mutex
	parallelism int
}

type bucket map[Key]bool
type buckets map[BucketKey]bucket

// NewThrottler returns a throttle that only runs `parallelism` items at once. When an item may need processing,
// `queue` is invoked.
func NewThrottler(parallelism int, bucketFunc BucketFunc, queue QueueFunc) Throttler {
	return &throttler{
		queue:       queue,
		bucketFunc:  bucketFunc,
		inProgress:  make(buckets),
		pending:     make(map[BucketKey]*priorityQueue),
		lock:        &sync.Mutex{},
		parallelism: parallelism,
	}
}

func (t *throttler) Init(wfs []wfv1.Workflow) error {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism == 0 {
		return nil
	}

	for _, wf := range wfs {
		key, err := cache.MetaNamespaceKeyFunc(&wf)
		if err != nil {
			return err
		}
		if wf.Status.Phase == wfv1.WorkflowRunning {
			bucketKey := t.bucketFunc(key)
			if _, ok := t.inProgress[bucketKey]; !ok {
				t.inProgress[bucketKey] = make(bucket)
			}
			t.inProgress[bucketKey][key] = true
		}
	}
	return nil
}

func (t *throttler) Add(key Key, priority int32, creationTime time.Time) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism == 0 {
		return
	}
	bucketKey := t.bucketFunc(key)
	if _, ok := t.pending[bucketKey]; !ok {
		t.pending[bucketKey] = &priorityQueue{itemByKey: make(map[string]*item)}
	}
	t.pending[bucketKey].add(key, priority, creationTime)
	t.queueThrottled(bucketKey)
}

func (t *throttler) Admit(key Key) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	if t.parallelism == 0 {
		return true
	}
	bucketKey := t.bucketFunc(key)
	if x, ok := t.inProgress[bucketKey]; ok && x[key] {
		return true
	}
	t.queueThrottled(bucketKey)
	return false
}

func (t *throttler) Remove(key Key) {
	t.lock.Lock()
	defer t.lock.Unlock()
	bucketKey := t.bucketFunc(key)
	if x, ok := t.inProgress[bucketKey]; ok {
		delete(x, key)
	}
	if x, ok := t.pending[bucketKey]; ok {
		x.remove(key)
	}
	t.queueThrottled(bucketKey)
}

func (t *throttler) queueThrottled(bucketKey BucketKey) {
	if _, ok := t.inProgress[bucketKey]; !ok {
		t.inProgress[bucketKey] = make(bucket)
	}
	inProgress := t.inProgress[bucketKey]
	pending, ok := t.pending[bucketKey]
	for ok && pending.Len() > 0 && t.parallelism > len(inProgress) {
		key := pending.pop().key
		inProgress[key] = true
		t.queue(key)
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

func (pq *priorityQueue) add(key Key, priority int32, creationTime time.Time) {
	if res, ok := pq.itemByKey[key]; ok {
		if res.priority != priority {
			res.priority = priority
			heap.Fix(pq, res.index)
		}
	} else {
		heap.Push(pq, &item{key: key, priority: priority, creationTime: creationTime})
	}
}

func (pq *priorityQueue) remove(key Key) {
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
