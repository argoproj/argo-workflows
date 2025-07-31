package sync

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	sema "golang.org/x/sync/semaphore"
)

type prioritySemaphore struct {
	name         string
	limitGetter  limitProvider
	pending      *priorityQueue
	semaphore    *sema.Weighted
	lockHolder   map[string]bool
	nextWorkflow NextWorkflow
	log          *log.Entry
}

var _ semaphore = &prioritySemaphore{}

func newInternalSemaphore(name string, nextWorkflow NextWorkflow, configMapGetter GetSyncLimit, syncLimitCacheTTL time.Duration) (*prioritySemaphore, error) {
	sem := &prioritySemaphore{
		name:         name,
		limitGetter:  newCachedLimit(configMapGetter, syncLimitCacheTTL),
		pending:      &priorityQueue{itemByKey: make(map[string]*item)},
		semaphore:    sema.NewWeighted(int64(0)),
		lockHolder:   make(map[string]bool),
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			"name":     name,
			"lockType": lockTypeSemaphore,
		}),
	}
	var err error
	limit := sem.getLimit()
	if limit == 0 {
		err = fmt.Errorf("failed to initialize semaphore %s with limit", name)
	}
	return sem, err
}

func (s *prioritySemaphore) getName() string {
	return s.name
}

func (s *prioritySemaphore) getLimit() int {
	limit, changed, err := s.limitGetter.get(s.name)
	if err != nil {
		s.log.WithError(err).Errorf("failed to get limit for semaphore %s", s.name)
		return 0
	}
	if changed {
		s.resize(limit)
	}
	return limit
}

func (s *prioritySemaphore) lock() bool {
	return true
}

func (s *prioritySemaphore) unlock() {}

func (s *prioritySemaphore) getCurrentPending() ([]string, error) {
	var keys []string
	for _, item := range s.pending.items {
		keys = append(keys, item.key)
	}
	return keys, nil
}

func (s *prioritySemaphore) getCurrentHolders() ([]string, error) {
	var keys []string
	for k := range s.lockHolder {
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *prioritySemaphore) resize(n int) bool {
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
	}
	return status
}

func (s *prioritySemaphore) release(key string) bool {
	limit := s.getLimit()
	if _, ok := s.lockHolder[key]; ok {
		delete(s.lockHolder, key)
		// When semaphore resized downward
		// Remove the excess holders from map once the done.
		if len(s.lockHolder) >= limit {
			return true
		}

		s.semaphore.Release(1)
		availableLocks := limit - len(s.lockHolder)
		s.log.Infof("Lock has been released by %s. Available locks: %d", key, availableLocks)
		if s.pending.Len() > 0 {
			s.notifyWaiters()
		}
	}
	return true
}

// notifyWaiters enqueues the next N workflows who are waiting for the semaphore to the workqueue,
// where N is the availability of the semaphore. If semaphore is out of capacity, this does nothing.
func (s *prioritySemaphore) notifyWaiters() {
	triggerCount := s.getLimit() - len(s.lockHolder)
	if s.pending.Len() < triggerCount {
		triggerCount = s.pending.Len()
	}
	for idx := 0; idx < triggerCount; idx++ {
		item := s.pending.items[idx]
		wfKey := workflowKey(item.key)
		s.log.Debugf("Enqueue the workflow %s", wfKey)
		s.nextWorkflow(wfKey)
	}
}

// workflowKey formulates the proper workqueue key given a semaphore queue item
func workflowKey(key string) string {
	parts := strings.Split(key, "/")
	if len(parts) == 3 {
		// the item is template semaphore (namespace/workflow-name/node-id) and so key must be
		// truncated to just: namespace/workflow-name
		return fmt.Sprintf("%s/%s", parts[0], parts[1])
	}
	return key
}

// addToQueue adds the holderkey into priority queue that maintains the priority order to acquire the lock.
func (s *prioritySemaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) error {
	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("Lock is already acquired by %s", holderKey)
		return nil
	}

	s.pending.add(holderKey, priority, creationTime)
	s.log.Debugf("Added into queue: %s", holderKey)
	return nil
}

func (s *prioritySemaphore) removeFromQueue(holderKey string) error {
	s.pending.remove(holderKey)
	s.log.Debugf("Removed from queue: %s", holderKey)
	return nil
}

func (s *prioritySemaphore) acquire(holderKey string, _ *transaction) bool {
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

// checkAcquire examines if tryAcquire would succeed
// returns
//
//	true, false if we would be able to take the lock
//	false, true if we already have the lock
//	false, false if the lock is not acquirable
//	string return is a user facing message when not acquirable
func (s *prioritySemaphore) checkAcquire(holderKey string, _ *transaction) (bool, bool, string) {
	limit := s.getLimit()
	if holderKey == "" {
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}

	if _, ok := s.lockHolder[holderKey]; ok {
		s.log.Debugf("%s is already holding a lock", holderKey)
		return false, true, ""
	}

	if limit == 0 {
		return false, false, fmt.Sprintf("Failed to get semaphore limit for %s", s.name)
	}

	waitingMsg := fmt.Sprintf("Waiting for %s lock. Lock status: %d/%d", s.name, limit-len(s.lockHolder), limit)

	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	if s.pending.Len() > 0 {
		item := s.pending.peek()
		if !isSameWorkflowNodeKeys(holderKey, item.key) {
			// Enqueue the front workflow if lock is available
			if len(s.lockHolder) < limit {
				s.nextWorkflow(workflowKey(item.key))
			}
			s.log.Infof("%s isn't at the front", holderKey)
			return false, false, waitingMsg
		}
	}
	if s.semaphore.TryAcquire(1) {
		s.semaphore.Release(1)
		return true, false, ""
	}

	s.log.Debugf("Current semaphore Holders. %v", s.lockHolder)
	return false, false, waitingMsg
}

func (s *prioritySemaphore) tryAcquire(holderKey string, tx *transaction) (bool, string) {
	acq, already, msg := s.checkAcquire(holderKey, tx)
	if already {
		return true, msg
	}
	if !acq {
		return false, msg
	}
	if s.acquire(holderKey, tx) {
		s.pending.pop()
		limit := s.getLimit()
		s.log.Infof("%s acquired by %s. Lock availability: %d/%d", s.name, holderKey, limit-len(s.lockHolder), limit)
		s.notifyWaiters()
		return true, ""
	}
	s.log.Debugf("Current semaphore Holders. %v", s.lockHolder)
	return false, msg
}

func (s *prioritySemaphore) probeWaiting() {}
