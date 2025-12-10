package sync

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
)

type databaseSemaphore struct {
	name         string
	limitGetter  limitProvider
	shortDBKey   string
	nextWorkflow NextWorkflow
	logger       loggerFn
	info         syncdb.DBInfo
	queries      syncdb.SyncQueries
	isMutex      bool
}

var _ semaphore = &databaseSemaphore{}

// newDatabaseSemaphore creates a database-backed semaphore with the given name, database key, next-workflow callback, and DB info.
// It initializes the semaphore state, cached limit provider, and query helper; it returns an error if the initial limit read from the database is zero.
func newDatabaseSemaphore(ctx context.Context, name string, dbKey string, nextWorkflow NextWorkflow, info syncdb.DBInfo, syncLimitCacheTTL time.Duration) (*databaseSemaphore, error) {
	logger := syncLogger{
		name:     name,
		lockType: lockTypeSemaphore,
	}
	sem := &databaseSemaphore{
		name:         name,
		shortDBKey:   dbKey,
		limitGetter:  nil,
		nextWorkflow: nextWorkflow,
		logger:       logger.get,
		info:         info,
		queries:      syncdb.NewSyncQueries(info.SessionProxy, info.Config),
		isMutex:      false,
	}
	sem.limitGetter = newCachedLimit(sem.getLimitFromDB, syncLimitCacheTTL)
	var err error
	limit := sem.getLimit(ctx)
	if limit == 0 {
		err = fmt.Errorf("failed to initialize semaphore %s with limit", name)
	}
	return sem, err
}

func (s *databaseSemaphore) longDBKey() string {
	if s.isMutex {
		return "mtx/" + s.shortDBKey
	}
	return "sem/" + s.shortDBKey
}

func (s *databaseSemaphore) getName() string {
	return s.name
}

func (s *databaseSemaphore) getLimitFromDB(ctx context.Context, _ string) (int, error) {
	logger := s.logger(ctx)
	// Update the limit from the database
	limit, err := s.queries.GetSemaphoreLimit(ctx, s.shortDBKey)
	if err != nil {
		logger.WithField("key", s.shortDBKey).WithError(err).Error(ctx, "Failed to get limit")
		return 0, err
	}
	logger.WithFields(logging.Fields{
		"limit": limit.SizeLimit,
		"key":   s.shortDBKey,
	}).Debug(ctx, "Current limit")
	return limit.SizeLimit, nil
}

// getLimit returns the semaphore limit. If isMutex this always returns 1.
// Otherwise queries the database for the limit.
func (s *databaseSemaphore) getLimit(ctx context.Context) int {
	logger := s.logger(ctx)
	logger.WithField("dbKey", s.shortDBKey).Info(ctx, "getLimit")
	limit, _, err := s.limitGetter.get(ctx, s.shortDBKey)
	if err != nil {
		logger.WithField("name", s.name).WithError(err).Error(ctx, "Failed to get limit")
		return 0
	}
	return limit
}

func (s *databaseSemaphore) currentState(ctx context.Context, sessionProxy *sqldb.SessionProxy, held bool) ([]string, error) {
	logger := s.logger(ctx)
	states, err := s.queries.GetCurrentState(ctx, sessionProxy, s.longDBKey(), held)
	if err != nil {
		logger.WithField("held", held).WithError(err).Error(ctx, "Failed to get current state")
		return nil, err
	}
	keys := make([]string, len(states))
	for i := range states {
		keys[i] = states[i].Key
	}
	return keys, nil
}

func (s *databaseSemaphore) getCurrentPending(ctx context.Context) ([]string, error) {
	return s.currentState(ctx, s.info.SessionProxy, false)
}

func (s *databaseSemaphore) getCurrentHolders(ctx context.Context) ([]string, error) {
	return s.currentHoldersSession(ctx, s.info.SessionProxy)
}

func (s *databaseSemaphore) currentHoldersSession(ctx context.Context, sessionProxy *sqldb.SessionProxy) ([]string, error) {
	return s.currentState(ctx, sessionProxy, true)
}

func (s *databaseSemaphore) lock(ctx context.Context) bool {
	logger := s.logger(ctx)
	// Check if lock already exists, in case we crashed and restarted
	existingLocks, err := s.queries.GetExistingLocks(ctx, s.longDBKey(), s.info.Config.ControllerName)

	if err == nil && len(existingLocks) > 0 {
		// Lock already exists
		logger.WithField("key", s.longDBKey()).Debug(ctx, "Lock already exists")
		return true
	}

	record := &syncdb.LockRecord{
		Name:       s.longDBKey(),
		Controller: s.info.Config.ControllerName,
		Time:       time.Now(),
	}
	err = s.queries.InsertLock(ctx, record)
	return err == nil
}

func (s *databaseSemaphore) unlock(ctx context.Context) {
	for {
		err := s.queries.DeleteLock(ctx, s.longDBKey())
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *databaseSemaphore) release(ctx context.Context, key string) bool {
	logger := s.logger(ctx)
	err := s.queries.ReleaseHeld(ctx, s.longDBKey(), key, s.info.Config.ControllerName)

	switch err {
	case nil:
		logger.WithField("key", key).Debug(ctx, "Released lock")
		s.notifyWaiters(ctx)
		return true
	default:
		logger.WithField("key", key).WithError(err).Error(ctx, "Failed to release lock")
		return false
	}
}

func (s *databaseSemaphore) queueOrdered(ctx context.Context, sessionProxy *sqldb.SessionProxy) ([]syncdb.StateRecord, error) {
	logger := s.logger(ctx)
	queue, err := s.queries.GetOrderedQueue(ctx, sessionProxy, s.longDBKey(), s.info.Config.InactiveControllerTimeout)
	if err != nil {
		logger.WithError(err).Error(ctx, "Failed to get ordered queue for semaphore notification")
		return nil, err
	}
	return queue, nil
}

// notifyWaiters enqueues the next N workflows who are waiting for the semaphore to the workqueue,
// where N is the availability of the semaphore. If semaphore is out of capacity, this does nothing.
func (s *databaseSemaphore) notifyWaiters(ctx context.Context) {
	logger := s.logger(ctx)
	limit := s.getLimit(ctx)
	// We don't need to run a transaction here, if we get it wrong it'll right itself
	holders, err := s.getCurrentHolders(ctx)
	if err != nil {
		logger.WithError(err).Error(ctx, "Failed to notify waiters")
		return
	}
	holdCount := len(holders)

	pending, err := s.queueOrdered(ctx, s.info.SessionProxy)
	if err != nil {
		return
	}
	triggerCount := min(limit-holdCount, len(pending))
	logger.WithFields(logging.Fields{
		"holdCount":    holdCount,
		"triggerCount": triggerCount,
		"pendingCount": len(pending),
	}).Debug(ctx, "Notifying waiters for semaphore")
	for idx := 0; idx < triggerCount; idx++ {
		item := pending[idx]
		if item.Controller != s.info.Config.ControllerName {
			continue
		}
		key := workflowKey(item.Key)
		logger.WithFields(logging.Fields{"key": item.Key, "workflowKey": key}).Debug(ctx, "Enqueueing workflow for semaphore notification")
		s.nextWorkflow(key)
	}
}

// addToQueue adds the holderkey into priority queue that maintains the priority order to acquire the lock.
func (s *databaseSemaphore) addToQueue(ctx context.Context, holderKey string, priority int32, creationTime time.Time) error {
	// Doesn't need a transaction, as no-one else should be inserting exactly this record ever
	states, err := s.queries.CheckQueueExists(ctx, s.longDBKey(), holderKey, s.info.Config.ControllerName)
	if err != nil {
		return err
	}
	if len(states) > 0 {
		return nil
	}
	record := &syncdb.StateRecord{
		Name:       s.longDBKey(),
		Key:        holderKey,
		Controller: s.info.Config.ControllerName,
		Held:       false,
		Priority:   priority,
		Time:       creationTime,
	}
	err = s.queries.AddToQueue(ctx, record)
	return err
}

func (s *databaseSemaphore) removeFromQueue(ctx context.Context, holderKey string) error {
	err := s.queries.RemoveFromQueue(ctx, s.longDBKey(), holderKey)
	return err
}

func (s *databaseSemaphore) checkAcquire(ctx context.Context, holderKey string, tx *transaction) (bool, bool, string) {
	logger := s.logger(ctx)
	if holderKey == "" {
		logger.WithFields(logging.Fields{
			"result":       false,
			"already_held": false,
			"message":      "bug: attempt to check semaphore with empty holder key",
		}).Info(ctx, "CheckAcquire failed")
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit(ctx)
	holders, err := s.currentHoldersSession(ctx, tx.sessionProxy)
	if err != nil {
		logger.WithFields(logging.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"error":        err.Error(),
		}).Info(ctx, "CheckAcquire failed")
		return false, false, err.Error()
	}
	if slices.Contains(holders, holderKey) {
		logger.WithFields(logging.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": true,
		}).Info(ctx, "CheckAcquire - already held")
		return false, true, ""
	}
	waitingMsg := fmt.Sprintf("Waiting for %s lock (%s). Lock status: %d/%d", s.name, s.longDBKey(), len(holders), limit)

	if len(holders) >= limit {
		logger.WithFields(logging.Fields{
			"key":             holderKey,
			"result":          false,
			"already_held":    false,
			"message":         waitingMsg,
			"current_holders": len(holders),
			"limit":           limit,
		}).Info(ctx, "CheckAcquire - limit exceeded")
		return false, false, waitingMsg
	}
	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	// Only live controllers are considered
	queue, err := s.queueOrdered(ctx, tx.sessionProxy)
	if err != nil {
		logger.WithFields(logging.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"error":        err.Error(),
		}).Info(ctx, "CheckAcquire failed")
		return false, false, err.Error()
	}
	if len(queue) == 0 {
		logger.WithFields(logging.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
		}).Info(ctx, "CheckAcquire - empty queue")
		return false, false, ""
	}
	if queue[0].Controller != s.info.Config.ControllerName {
		logger.WithFields(logging.Fields{
			"key":                holderKey,
			"result":             false,
			"already_held":       false,
			"message":            waitingMsg,
			"queue_controller":   queue[0].Controller,
			"current_controller": s.info.Config.ControllerName,
		}).Info(ctx, "CheckAcquire - different controller")
		return false, false, waitingMsg
	}
	if !isSameWorkflowNodeKeys(holderKey, queue[0].Key) {
		// Enqueue the queue[0] workflow if lock is available
		if len(holders) < limit {
			s.nextWorkflow(queue[0].Key)
		}
		logger.WithFields(logging.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"message":      waitingMsg,
			"queue_key":    queue[0].Key,
		}).Info(ctx, "CheckAcquire - not first in queue")
		return false, false, waitingMsg
	}
	logger.WithFields(logging.Fields{
		"key":          holderKey,
		"result":       true,
		"already_held": false,
	}).Info(ctx, "CheckAcquire - can acquire")
	return true, false, ""
}

func (s *databaseSemaphore) acquire(ctx context.Context, holderKey string, tx *transaction) bool {
	logger := s.logger(ctx)
	limit := s.getLimit(ctx)
	existing, err := s.currentHoldersSession(ctx, tx.sessionProxy)
	if err != nil {
		logger.WithField("key", holderKey).WithError(err).Error(ctx, "Failed to acquire lock")
		return false
	}
	if len(existing) < limit {
		pending, err := s.queries.GetPendingInQueueWithSession(ctx, tx.sessionProxy, s.longDBKey(), holderKey, s.info.Config.ControllerName)
		if err != nil {
			logger.WithField("key", holderKey).WithError(err).Error(ctx, "Failed to acquire lock")
			return false
		}
		if len(pending) > 0 {
			err := s.queries.UpdateStateToHeldWithSession(ctx, tx.sessionProxy, s.longDBKey(), holderKey, s.info.Config.ControllerName)
			if err != nil {
				logger.WithField("key", holderKey).WithError(err).Error(ctx, "Failed to acquire lock")
				return false
			}
		} else {
			record := &syncdb.StateRecord{
				Name:       s.longDBKey(),
				Key:        holderKey,
				Controller: s.info.Config.ControllerName,
				Held:       true,
			}
			err := s.queries.InsertHeldStateWithSession(ctx, tx.sessionProxy, record)
			if err != nil {
				logger.WithField("key", holderKey).WithError(err).Error(ctx, "Failed to acquire lock")
				return false
			}
		}
		logger.WithFields(logging.Fields{
			"key":    holderKey,
			"result": true,
		}).Info(ctx, "Acquire succeeded")
		return true
	}
	logger.WithFields(logging.Fields{
		"key":             holderKey,
		"result":          false,
		"reason":          "limit exceeded",
		"current_holders": len(existing),
		"limit":           limit,
	}).Info(ctx, "Acquire failed")
	return false
}

func (s *databaseSemaphore) tryAcquire(ctx context.Context, holderKey string, tx *transaction) (bool, string) {
	logger := s.logger(ctx)
	acq, already, msg := s.checkAcquire(ctx, holderKey, tx)
	if already {
		logger.WithFields(logging.Fields{
			"key":     holderKey,
			"result":  true,
			"message": msg,
		}).Info(ctx, "tryAcquire - already held")
		return true, msg
	}
	if !acq {
		logger.WithFields(logging.Fields{
			"key":     holderKey,
			"result":  false,
			"message": msg,
		}).Info(ctx, "tryAcquire - cannot acquire")
		return false, msg
	}
	if s.acquire(ctx, holderKey, tx) {
		logger.WithFields(logging.Fields{
			"key":    holderKey,
			"result": true,
		}).Info(ctx, "tryAcquire succeeded")
		s.notifyWaiters(ctx)
		return true, ""
	}
	logger.WithFields(logging.Fields{
		"key":     holderKey,
		"result":  false,
		"message": msg,
	}).Info(ctx, "tryAcquire failed")
	return false, msg
}

func (s *databaseSemaphore) expireLocks(ctx context.Context) {
	logger := s.logger(ctx)
	rowsAffected, err := s.queries.ExpireInactiveLocks(ctx, s.info.Config.InactiveControllerTimeout)
	if err != nil {
		logger.WithError(err).Error(ctx, "Failed to expire locks")
	} else if rowsAffected > 0 {
		logger.WithField("rowsAffected", rowsAffected).Info(ctx, "Expired locks")
	}
}

func (s *databaseSemaphore) probeWaiting(ctx context.Context) {
	s.notifyWaiters(ctx)
	s.expireLocks(ctx)
}