package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	sema "golang.org/x/sync/semaphore"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type prioritySemaphore struct {
	name         string
	limitGetter  limitProvider
	pending      *priorityQueue
	semaphore    *sema.Weighted
	lockHolder   map[string]bool
	nextWorkflow NextWorkflow
	logger       loggerFn
}

var _ semaphore = &prioritySemaphore{}

func newInternalSemaphore(ctx context.Context, name string, nextWorkflow NextWorkflow, configMapGetter GetSyncLimit, syncLimitCacheTTL time.Duration) (*prioritySemaphore, error) {
	logger := syncLogger{
		name:     name,
		lockType: lockTypeSemaphore,
	}
	sem := &prioritySemaphore{
		name:         name,
		limitGetter:  newCachedLimit(configMapGetter, syncLimitCacheTTL),
		pending:      &priorityQueue{itemByKey: make(map[string]*item)},
		semaphore:    sema.NewWeighted(int64(0)),
		lockHolder:   make(map[string]bool),
		nextWorkflow: nextWorkflow,
		logger:       logger.get,
	}
	var err error
	limit := sem.getLimit(ctx)
	if limit == 0 {
		err = fmt.Errorf("failed to initialize semaphore %s with limit", name)
	}
	return sem, err
}

func (s *prioritySemaphore) getName() string {
	return s.name
}

func (s *prioritySemaphore) getLimit(ctx context.Context) int {
	limit, changed, err := s.limitGetter.get(ctx, s.name)
	if err != nil {
		s.logger(ctx).WithError(err).WithField("name", s.name).Error(ctx, "failed to get limit for semaphore")
		return 0
	}
	if changed {
		s.resize(ctx, limit)
	}
	return limit
}

func (s *prioritySemaphore) lock(_ context.Context) bool {
	return true
}

func (s *prioritySemaphore) unlock(_ context.Context) {}

func (s *prioritySemaphore) getCurrentPending(_ context.Context) ([]string, error) {
	var keys []string
	for _, item := range s.pending.items {
		keys = append(keys, item.key)
	}
	return keys, nil
}

func (s *prioritySemaphore) getCurrentHolders(_ context.Context) ([]string, error) {
	var keys []string
	for k := range s.lockHolder {
		keys = append(keys, k)
	}
	return keys, nil
}

func (s *prioritySemaphore) resize(ctx context.Context, n int) bool {
	// downward case, acquired n locks
	cur := min(len(s.lockHolder), n)

	semaphore := sema.NewWeighted(int64(n))
	status := semaphore.TryAcquire(int64(cur))
	if status {
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"name": s.name,
			"from": cur,
			"to":   n,
		}).Info(ctx, "semaphore resized")
		s.semaphore = semaphore
	}
	return status
}

func (s *prioritySemaphore) release(ctx context.Context, key string) bool {
	limit := s.getLimit(ctx)
	if _, ok := s.lockHolder[key]; ok {
		delete(s.lockHolder, key)
		// When semaphore resized downward
		// Remove the excess holders from map once the done.
		if len(s.lockHolder) >= limit {
			return true
		}

		s.semaphore.Release(1)
		availableLocks := limit - len(s.lockHolder)
		logging.RequireLoggerFromContext(ctx).WithFields(logging.Fields{
			"key":            key,
			"availableLocks": availableLocks,
		}).Info(ctx, "Lock has been released")
		if s.pending.Len() > 0 {
			s.notifyWaiters(ctx)
		}
	}
	return true
}

// notifyWaiters enqueues the next N workflows who are waiting for the semaphore to the workqueue,
// where N is the availability of the semaphore. If semaphore is out of capacity, this does nothing.
func (s *prioritySemaphore) notifyWaiters(ctx context.Context) {
	triggerCount := min(s.pending.Len(), s.getLimit(ctx)-len(s.lockHolder))
	for idx := 0; idx < triggerCount; idx++ {
		item := s.pending.items[idx]
		wfKey := workflowKey(item.key)
		s.logger(ctx).WithField("workflow", wfKey).Debug(ctx, "Enqueue the workflow")
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
func (s *prioritySemaphore) addToQueue(ctx context.Context, holderKey string, priority int32, creationTime time.Time) error {
	logger := s.logger(ctx)

	if _, ok := s.lockHolder[holderKey]; ok {
		logger.WithField("holderKey", holderKey).Debug(ctx, "Lock is already acquired")
		return nil
	}

	s.pending.add(holderKey, priority, creationTime)
	logger.WithField("holderKey", holderKey).Debug(ctx, "Added into queue")
	return nil
}

func (s *prioritySemaphore) removeFromQueue(ctx context.Context, holderKey string) error {
	logger := s.logger(ctx)
	s.pending.remove(holderKey)
	logger.WithField("holderKey", holderKey).Debug(ctx, "Removed from queue")
	return nil
}

func (s *prioritySemaphore) acquire(_ context.Context, holderKey string, _ *transaction) bool {
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
func (s *prioritySemaphore) checkAcquire(ctx context.Context, holderKey string, _ *transaction) (bool, bool, string) {
	logger := s.logger(ctx)
	limit := s.getLimit(ctx)
	if holderKey == "" {
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}

	if _, ok := s.lockHolder[holderKey]; ok {
		logger.WithField("holderKey", holderKey).Debug(ctx, "is already holding a lock")
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
			logger.WithField("holderKey", holderKey).Info(ctx, "isn't at the front")
			return false, false, waitingMsg
		}
	}
	if s.semaphore.TryAcquire(1) {
		s.semaphore.Release(1)
		return true, false, ""
	}

	logger.WithField("lockHolder", s.lockHolder).Debug(ctx, "Current semaphore Holders")
	return false, false, waitingMsg
}

func (s *prioritySemaphore) tryAcquire(ctx context.Context, holderKey string, tx *transaction) (bool, string) {
	logger := s.logger(ctx)
	acq, already, msg := s.checkAcquire(ctx, holderKey, tx)
	if already {
		return true, msg
	}
	if !acq {
		return false, msg
	}
	if s.acquire(ctx, holderKey, tx) {
		s.pending.pop()
		limit := s.getLimit(ctx)
		logger.WithFields(logging.Fields{
			"name":      s.name,
			"holderKey": holderKey,
			"available": limit - len(s.lockHolder),
			"limit":     limit,
		}).Info(ctx, "acquired")
		s.notifyWaiters(ctx)
		return true, ""
	}
	logger.WithField("lockHolder", s.lockHolder).Debug(ctx, "Current semaphore Holders")
	return false, msg
}

func (s *prioritySemaphore) probeWaiting(_ context.Context) {}
