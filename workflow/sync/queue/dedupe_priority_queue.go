package queue

type Keyed interface {
	Prioritizable
	GetKey() string
}

// A priority queue that **helps** de-dupe items based on the key.
// Additionally, this add func that take key rather than interface{} as the argument.
type DedupePriorityQueue struct {
	PriorityQueue
	known map[string]*Item
}

func NewDedupePriorityQueue() *DedupePriorityQueue {
	return &DedupePriorityQueue{known: make(map[string]*Item)}
}

// You must not push an Item with out first doing the following:
//
// if v, ok := pq.Contains(x) ; ok{heap.Remove(pq, v.(*Item).index)}
//
func (pq *DedupePriorityQueue) Push(x interface{}) {
	item := x.(*Item)
	k := item.Value.(Keyed)
	if pq.Contains(k.GetKey()) {
		panic("queue already contains Item, programming error")
	}
	pq.PriorityQueue.Push(x)
	pq.known[k.GetKey()] = item
}

func (pq *DedupePriorityQueue) Pop() interface{} {
	x := pq.PriorityQueue.Pop().(*Item)
	delete(pq.known, x.Value.(Keyed).GetKey())
	return x
}

func (pq *DedupePriorityQueue) Index(key string) int {
	return pq.known[key].index
}

func (pq *DedupePriorityQueue) Contains(key string) bool {
	_, contains := pq.known[key]
	return contains
}
