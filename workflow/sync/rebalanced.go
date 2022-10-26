package sync

import (
	"math"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// TODO: maintain cache, expire entries

// RebalanceQueue implements the rebalance strategy for consumption in Semaphore
type RebalanceQueue struct {
	items             []*item           // standard item with an attached rebalance key
	rebalanceKeyCache map[string]string // mapping from resource name (holder) to holder's rebalance key
	semaphore         Semaphore         // reference to parent semaphore, so we can get holders
}

func NewRebalanceQueue() *RebalanceQueue {
	return &RebalanceQueue{
		items:             make([]*item, 0),
		rebalanceKeyCache: make(map[string]string, 0),
	}
}

// RebalanceQueue needs to reference semaphore to know current lock holders
func (r *RebalanceQueue) setParentSemaphore(s Semaphore) {
	r.semaphore = s
}

// determines the best ordering based on currently-outstanding keys
func (r *RebalanceQueue) onRelease(key Key) {
	// we need pending + holding counts for rebalance keys, and just holding counts
	allRebalanceKeys := make(map[string]int, 0)
	holderRebalanceKeys := make(map[string]int, 0)

	for _, h := range r.semaphore.getCurrentHolders() {
		rk := r.rebalanceKeyCache[h]
		allRebalanceKeys[rk] += 1
		if h != key {
			holderRebalanceKeys[rk] += 1
		}
	}

	for _, p := range r.items {
		rk := r.rebalanceKeyCache[p.key]
		allRebalanceKeys[rk] += 1
	}

	// partition r.queue into items that can / can't be scheduled
	can := make([]*item, 0)
	overflow := make(map[string]*item, 0)
	cant := make([]*item, 0)

	maxLocksPerUserFloor := math.Floor(float64(r.semaphore.getLimit()) / float64(len(allRebalanceKeys)))
	// iterate through pending queue to determine what has changed
	for _, p := range r.items {
		if p.key != key {
			rebalanceKey := r.rebalanceKeyCache[p.key]
			// we want to ignore "key" since removal from queue happens after release in semaphore
			if float64(holderRebalanceKeys[rebalanceKey]) < maxLocksPerUserFloor {
				can = append(can, p)
				holderRebalanceKeys[rebalanceKey] += 1
			} else {
				// don't add to overflow if already maxLocksPerUserFloor + 1 because we run the risk of kicking
				// off an extra one!
				if overflow[rebalanceKey] == nil && holderRebalanceKeys[rebalanceKey] < int(maxLocksPerUserFloor)+1 {
					overflow[rebalanceKey] = p
				} else {
					cant = append(cant, p)
				}
			}
		}
	}

	// fill "can" with earliest submitted items until we reach limit
	for len(overflow) > 0 {
		var earliest *item
		var rebalanceKey string
		for k, p := range overflow {
			if earliest == nil || earliest.creationTime.Unix() > p.creationTime.Unix() {
				earliest = p
				rebalanceKey = k
			}
		}
		delete(overflow, rebalanceKey)
		can = append(can, earliest)
	}

	delete(r.rebalanceKeyCache, key)

	r.items = append(can, cant...)
}

func (r *RebalanceQueue) peek() *item {
	return r.items[0]
}

func (r *RebalanceQueue) pop() *item {
	first := r.items[0]
	r.remove(first.key)
	return first
}

// rebalance queues, at the moment, do not support priority
//
// if queue has key, skip
// if queue doesn't, add along with rebalance key
func (r *RebalanceQueue) add(key Key, _ int32, creationTime time.Time, syncLockRef *wfv1.Synchronization) {
	found := false
	for _, p := range r.items {
		if p.key == key {
			found = true
		}
	}

	if !found {
		rebalanceKey := ""
		if syncLockRef != nil && syncLockRef.Semaphore != nil && syncLockRef.Semaphore.RebalanceKey != nil {
			rebalanceKey = *syncLockRef.Semaphore.RebalanceKey
		}

		r.rebalanceKeyCache[key] = rebalanceKey

		r.items = append(r.items, &item{key: key, creationTime: creationTime, priority: 1, index: -1})
	}
}

func (r *RebalanceQueue) remove(key Key) {
	for i, p := range r.items {
		if p.key == key {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return
		}
	}
}

func (r *RebalanceQueue) Len() int {
	return len(r.items)
}

func (r *RebalanceQueue) all() []*item {
	return r.items
}
