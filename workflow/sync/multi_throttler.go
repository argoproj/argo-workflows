package sync

import (
	"container/heap"
	"os"
	"sync"
	"time"

	"k8s.io/client-go/tools/cache"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var logg *log.Logger

func init() {
	logg = log.New()
	f, err := os.OpenFile("./dump.txt", os.O_WRONLY|os.O_CREATE, 0666)
	if err == nil {
		logg.SetOutput(f)
	}
}

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
	Debug()
}

type Key = string
type QueueFunc func(Key)

// NewMultiThrottler creates a new multi throttler for throttling both namespace and global parallelism, a parallelism value of zero disables throttling
func NewMultiThrottler(parallelism int, namespaceParallelism map[string]int, namespaceParallelismDefault int, queue QueueFunc) Throttler {
	return &multiThrottler{
		queue:                       queue,
		namespaceParallelism:        namespaceParallelism,
		namespaceParallelismDefault: namespaceParallelismDefault,
		totalParallelism:            parallelism,
		running:                     make(map[Key]bool),
		pending:                     make(map[string]*priorityQueue),
		lock:                        &sync.Mutex{},
	}
}

type multiThrottler struct {
	queue                       QueueFunc
	namespaceParallelism        map[string]int
	namespaceParallelismDefault int
	totalParallelism            int
	running                     map[Key]bool
	pending                     map[string]*priorityQueue
	lock                        *sync.Mutex
}

func (m *multiThrottler) Init(wfs []wfv1.Workflow) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	type keyNamespacePair struct {
		key       string
		namespace string
	}
	pairs := []keyNamespacePair{}
	for _, wf := range wfs {
		if wf.Status.Phase != wfv1.WorkflowRunning {
			continue
		}
		key, err := cache.MetaNamespaceKeyFunc(&wf)
		if err != nil {
			return err
		}
		namespace, _, err := cache.SplitMetaNamespaceKey(key)
		if err != nil {
			return err
		}
		pairs = append(pairs, keyNamespacePair{key: key, namespace: namespace})
	}

	for _, pair := range pairs {
		m.running[pair.key] = true
	}
	return nil
}

func (m *multiThrottler) namespaceCount(namespace string) (int, int) {
	setLimit, has := m.namespaceParallelism[namespace]
	if !has {
		m.namespaceParallelism[namespace] = m.namespaceParallelismDefault
		setLimit = m.namespaceParallelismDefault
	}
	count := 0
	for key := range m.running {
		ns, _, _ := cache.SplitMetaNamespaceKey(key)
		if ns == namespace {
			count++
		}
	}
	return count, setLimit
}

func (m *multiThrottler) namespaceAllows(namespace string) bool {
	count, limit := m.namespaceCount(namespace)
	return count < limit || limit == 0
}

func (m *multiThrottler) Add(key Key, priority int32, creationTime time.Time) {
	m.lock.Lock()
	defer m.lock.Unlock()
	logg.Infof("[DEBUG][ADD] on key: %s", key)
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}
	_, ok := m.pending[namespace]
	if !ok {
		m.pending[namespace] = &priorityQueue{itemByKey: make(map[string]*item)}
	}

	m.pending[namespace].add(key, priority, creationTime)
	m.queueThrottled()
}

func (m *multiThrottler) Admit(key Key) bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	logg.Infof("[DEBUG][ADMIT] on key: %s", key)

	_, ok := m.running[key]
	if ok {
		return true
	}
	m.queueThrottled()
	return false
}

func (m *multiThrottler) Remove(key Key) {
	m.lock.Lock()
	defer m.lock.Unlock()
	logg.Infof("[DEBUG][REMOVE] on key: %s", key)

	namespace, _, _ := cache.SplitMetaNamespaceKey(key)
	if _, ok := m.running[key]; ok {
		delete(m.running, key)
	}
	m.pending[namespace].remove(key)
	m.queueThrottled()
}

func (m *multiThrottler) Debug() {
	m.lock.Lock()
	defer m.lock.Unlock()
	cnts := make(map[string]int)
	logg.Infof("[DEBUG][CONSOLE] total parallelism: %d, total count: %d", m.totalParallelism, len(m.running))
	log.Infof("[DEBUG][CONSOLE] total parallelism: %d, total count: %d", m.totalParallelism, len(m.running))

	for key := range m.running {
		logg.Infof("[DEBUG][RUNNING] %s", key)
		log.Infof("[DEBUG][RUNNING] %s", key)
		namespace, _, _ := cache.SplitMetaNamespaceKey(key)
		cnts[namespace] = cnts[namespace] + 1
	}

	for n, cnt := range cnts {
		logg.Infof("[DEBUG][NAMESPACE] %s\t%d", n, cnt)
		log.Infof("[DEBUG][NAMESPACE] %s\t%d", n, cnt)
	}
}

func (m *multiThrottler) queueThrottled() {
	if m.totalParallelism != 0 && len(m.running) >= m.totalParallelism {
		return
	}
	var bestItem *item
	var oldPq *priorityQueue

	cnts := make(map[string]int)

	for key := range m.running {
		namespace, _, _ := cache.SplitMetaNamespaceKey(key)
		cnts[namespace] = cnts[namespace] + 1
	}

	for _, pq := range m.pending {
		if len(pq.items) == 0 {
			continue
		}
		currItem := pq.peek()

		namespace, _, err := cache.SplitMetaNamespaceKey(currItem.key)
		if err != nil {
			return
		}
		if !m.namespaceAllows(namespace) {
			logg.Infof("[DEBUG][queueThrottled] %s was not allowed in namespace %s, count: %d", currItem.key, namespace, cnts[namespace])
			continue
		}
		if bestItem == nil {
			bestItem = currItem
			oldPq = pq
			continue
		}

		if LessByItem(bestItem, currItem) {
			bestItem = currItem
			oldPq = pq
		}
	}

	if bestItem != nil {
		sameKey := oldPq.pop()
		if sameKey.key != bestItem.key {
			panic("unreachable code was reached")
		}
		m.running[bestItem.key] = true
		m.queue(bestItem.key)
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
	return LessByItem(pq.items[i], pq.items[j])
}

func LessByItem(i *item, j *item) bool {
	if i.priority == j.priority {
		return i.creationTime.Before(j.creationTime)
	}
	return i.priority > j.priority
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
