package sync

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	sema "golang.org/x/sync/semaphore"
)

type PrioritySemaphore struct {
	name         string
	limit        int
	pending      *priorityQueue
	semaphore    *sema.Weighted
	lockHolder   map[string]bool
	lock         *sync.Mutex
	nextWorkflow NextWorkflow
	log          *log.Entry
}

var _ Semaphore = &PrioritySemaphore{}

func NewSemaphore(name string, limit int, nextWorkflow NextWorkflow, lockType string) *PrioritySemaphore {
	return &PrioritySemaphore{
		name:         name,
		limit:        limit,
		pending:      &priorityQueue{itemByKey: make(map[string]*item)},
		semaphore:    sema.NewWeighted(int64(limit)),
		lockHolder:   make(map[string]bool),
		lock:         &sync.Mutex{},
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			lockType: name,
		}),
	}
}

func (s *PrioritySemaphore) getName() string {
	return s.name
}

func (s *PrioritySemaphore) getLimit() int {
	return s.limit
}

func (s *PrioritySemaphore) getCurrentPending() []string {
	var keys []string
	for _, item := range s.pending.items {
		keys = append(keys, item.key)
	}
	return keys
}

func (s *PrioritySemaphore) getCurrentHolders() []string {
	var keys []string
	for k := range s.lockHolder {
		keys = append(keys, k)
	}
	return keys
}

func (s *PrioritySemaphore) resize(n int) bool {
	s.lock.Lock()
	defer s.lock.Unlock()

	cur := len(s.lockHolder)
	// downward case, acquired n locks
	if cur > n {
		cur = n
	}

	semaphore := sema.NewWeighted(int64(n))
	status := semaphore.TryAcquire(int64(cur))
	if status {
		s.log.Infof("%s semaphore resized from %d to %d", s.name, cur, n)
		s.semaphore = semaphore
		s.limit = n
	}
	return status
}

func (s *PrioritySemaphore) release(key string) bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	if _, ok := s.lockHolder[key]; ok {
		delete(s.lockHolder, key)
		// When semaphore resized downward
		// Remove the excess holders from map once the done.
		if len(s.lockHolder) >= s.limit {
			return true
		}

		s.semaphore.Release(1)
		availableLocks := s.limit - len(s.lockHolder)
		s.log.Infof("Lock has been released by %s. Available locks: %d", key, availableLocks)
		if s.pending.Len() > 0 {
			s.notifyWaiters()
		}
	}
	return true
}

// notifyWaiters enqueues the next N workflows who are waiting for the semaphore to the workqueue,
// where N is the availability of the semaphore. If semaphore is out of capacity, this does nothing.
func (s *PrioritySemaphore) notifyWaiters() {
	triggerCount := s.limit - len(s.lockHolder)
	if s.pending.Len() < triggerCount {
		triggerCount = s.pending.Len()
	}
	for idx := 0; idx < triggerCount; idx++ {
		item := s.pending.items[idx]
		wfKey := workflowKey(item)
		s.log.Debugf("Enqueue the workflow %s", wfKey)
		s.nextWorkflow(wfKey)
	}
}

// workflowKey formulates the proper workqueue key given a semaphore queue item
func workflowKey(i *item) string {
	parts := strings.Split(i.key, "/")
	if len(parts) == 3 {
		// the item is template semaphore (namespace/workflow-name/node-id) and so key must be
		// truncated to just: namespace/workflow-name
		return fmt.Sprintf("%s/%s", parts[0], parts[1])
	}
	return i.key
}

// addToQueue adds the holderkey into priority queue that maintains the priority order to acquire the lock.
func (s *PrioritySemaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("Lock is already acquired by %s", holderKey)
		return
	}

	s.pending.add(holderKey, priority, creationTime)
	s.log.Debugf("Added into queue: %s", holderKey)
}

func (s *PrioritySemaphore) removeFromQueue(holderKey string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.pending.remove(holderKey)
	s.log.Debugf("Removed from queue: %s", holderKey)
}

func (s *PrioritySemaphore) acquire(holderKey string) bool {
	if s.semaphore.TryAcquire(1) {
		s.lockHolder[holderKey] = true
		return true
	}
	return false
}

func isSameWorkflowNodeKeys(firstKey, secondKey string) bool {
	firstItems := strings.Split(firstKey, "/")
	secondItems := strings.Split(secondKey, "/")

	if len(firstItems) != len(secondItems) {
		return false
	}
	// compare workflow name
	return firstItems[1] == secondItems[1]
}

func (s *PrioritySemaphore) tryAcquire(holderKey string) (bool, string) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("%s is already holding a lock", holderKey)
		return true, ""
	}
	var nextKey string

	waitingMsg := fmt.Sprintf("Waiting for %s lock. Lock status: %d/%d", s.name, s.limit-len(s.lockHolder), s.limit)

	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	if s.pending.Len() > 0 {
		item := s.pending.peek()
		if holderKey != nextKey && !isSameWorkflowNodeKeys(holderKey, item.key) {
			// Enqueue the front workflow if lock is available
			if len(s.lockHolder) < s.limit {
				s.nextWorkflow(workflowKey(item))
			}
			return false, waitingMsg
		}
	}

	if s.acquire(holderKey) {
		s.pending.pop()
		s.log.Infof("%s acquired by %s. Lock availability: %d/%d", s.name, holderKey, s.limit-len(s.lockHolder), s.limit)
		s.notifyWaiters()
		return true, ""
	}
	s.log.Debugf("Current semaphore Holders. %v", s.lockHolder)
	return false, waitingMsg
}
