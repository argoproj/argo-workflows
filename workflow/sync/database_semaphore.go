package sync

import (
	"fmt"
	"slices"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
)

type databaseSemaphore struct {
	name         string
	limitGetter  limitProvider
	shortDbKey   string
	nextWorkflow NextWorkflow
	log          *log.Entry
	info         dbInfo
	isMutex      bool
}

type limitRecord struct {
	Name      string `db:"name"`
	SizeLimit int    `db:"sizelimit"`
}

type stateRecord struct {
	Name       string    `db:"name"`        // semaphore name identifier
	Key        string    `db:"workflowkey"` // workflow key holding or waiting for the lock of the form <namespace>/<name>
	Controller string    `db:"controller"`  // controller where the workflow is running
	Held       bool      `db:"held"`
	Priority   int32     `db:"priority"` // higher number = higher priority in queue
	Time       time.Time `db:"time"`     // timestamp of creation or last update
}

type controllerHealthRecord struct {
	Controller string    `db:"controller"` // controller where the workflow is running
	Time       time.Time `db:"time"`       // timestamp of creation or last update
}

type lockRecord struct {
	Name       string    `db:"name"`       // semaphore name identifier
	Controller string    `db:"controller"` // controller where the workflow is running
	Time       time.Time `db:"time"`       // timestamp of creation
}

const (
	limitNameField = "name"
	limitSizeField = "sizelimit"

	stateNameField       = "name"
	stateKeyField        = "workflowkey"
	stateControllerField = "controller"
	stateHeldField       = "held"
	statePriorityField   = "priority"
	stateTimeField       = "time"

	controllerNameField = "controller"
	controllerTimeField = "time"

	lockNameField       = "name"
	lockControllerField = "controller"
	lockTimeField       = "time"
)

var _ semaphore = &databaseSemaphore{}

func newDatabaseSemaphore(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo, syncLimitCacheTTL time.Duration) (*databaseSemaphore, error) {
	sem := &databaseSemaphore{
		name:         name,
		shortDbKey:   dbKey,
		limitGetter:  nil,
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeSemaphore,
			"name":     name,
		}),
		info:    info,
		isMutex: false,
	}
	sem.limitGetter = newCachedLimit(sem.getLimitFromDB, syncLimitCacheTTL)
	var err error
	limit := sem.getLimit()
	if limit == 0 {
		err = fmt.Errorf("failed to initialize semaphore %s with limit", name)
	}
	return sem, err
}

func (s *databaseSemaphore) longDbKey() string {
	if s.isMutex {
		return "mtx/" + s.shortDbKey
	}
	return "sem/" + s.shortDbKey
}

func (s *databaseSemaphore) getName() string {
	return s.name
}

func (s *databaseSemaphore) getLimitFromDB(_ string) (int, error) {
	// Update the limit from the database
	limit := &limitRecord{}
	err := s.info.session.SQL().
		Select(limitSizeField).
		From(s.info.config.limitTable).
		Where(db.Cond{limitNameField: s.shortDbKey}).
		One(limit)
	if err != nil {
		s.log.WithField("key", s.shortDbKey).WithError(err).Error("Failed to get limit")
		return 0, err
	}
	s.log.WithFields(log.Fields{
		"limit": limit.SizeLimit,
		"key":   s.shortDbKey,
	}).Debug("Current limit")
	return limit.SizeLimit, nil
}

// getLimit returns the semaphore limit. If isMutex this always returns 1.
// Otherwise queries the database for the limit.
func (s *databaseSemaphore) getLimit() int {
	log.WithFields(log.Fields{
		"dbKey": s.shortDbKey,
	}).Infof("getLimit")
	limit, _, err := s.limitGetter.get(s.shortDbKey)
	if err != nil {
		s.log.WithError(err).Errorf("Failed to get limit for semaphore %s", s.name)
		return 0
	}
	return limit
}

func (s *databaseSemaphore) currentState(session db.Session, held bool) ([]string, error) {
	var states []stateRecord
	err := session.SQL().
		Select(stateKeyField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateHeldField: held}).
		And(db.Cond{stateNameField: s.longDbKey()}).
		All(&states)
	if err != nil {
		s.log.WithField("held", held).WithError(err).Error("Failed to get current state")
		return nil, err
	}
	keys := make([]string, len(states))
	for i := range states {
		keys[i] = states[i].Key
	}
	return keys, nil
}

func (s *databaseSemaphore) getCurrentPending() ([]string, error) {
	return s.currentState(s.info.session, false)
}

func (s *databaseSemaphore) getCurrentHolders() ([]string, error) {
	return s.currentHoldersSession(s.info.session)
}

func (s *databaseSemaphore) currentHoldersSession(session db.Session) ([]string, error) {
	return s.currentState(session, true)
}

func (s *databaseSemaphore) lock() bool {
	// Check if lock already exists, in case we crashed and restarted
	var existingLocks []lockRecord
	err := s.info.session.SQL().
		Select(lockNameField).
		From(s.info.config.lockTable).
		Where(db.Cond{lockNameField: s.longDbKey()}).
		And(db.Cond{lockControllerField: s.info.config.controllerName}).
		All(&existingLocks)

	if err == nil && len(existingLocks) > 0 {
		// Lock already exists
		s.log.WithField("key", s.longDbKey()).Debug("Lock already exists")
		return true
	}

	record := &lockRecord{
		Name:       s.longDbKey(),
		Controller: s.info.config.controllerName,
		Time:       time.Now(),
	}
	_, err = s.info.session.Collection(s.info.config.lockTable).Insert(record)
	return err == nil
}

func (s *databaseSemaphore) unlock() {
	for {
		_, err := s.info.session.SQL().
			DeleteFrom(s.info.config.lockTable).
			Where(db.Cond{lockNameField: s.longDbKey()}).
			Exec()
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *databaseSemaphore) release(key string) bool {
	_, err := s.info.session.SQL().
		DeleteFrom(s.info.config.stateTable).
		Where(db.Cond{stateHeldField: true}).
		And(db.Cond{stateNameField: s.longDbKey()}).
		And(db.Cond{stateKeyField: key}).
		And(db.Cond{stateControllerField: s.info.config.controllerName}).
		Exec()

	switch err {
	case nil:
		s.log.WithField("key", key).Debug("Released lock")
		s.notifyWaiters()
		return true
	default:
		s.log.WithField("key", key).WithError(err).Error("Failed to release lock")
		return false
	}
}

func (s *databaseSemaphore) queueOrdered(session db.Session) ([]stateRecord, error) {
	since := time.Now().Add(-s.info.config.inactiveControllerTimeout)
	var queue []stateRecord
	subquery := session.SQL().
		Select(controllerNameField).
		From(s.info.config.controllerTable).
		And(db.Cond{controllerTimeField + " >": since})

	err := session.SQL().
		Select(stateKeyField, stateControllerField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.longDbKey()}).
		And(db.Cond{stateHeldField: false}).
		And(db.Cond{
			"controller IN": subquery,
		}).
		OrderBy(statePriorityField+" DESC", stateTimeField+" ASC").
		All(&queue)

	if err != nil {
		s.log.WithError(err).Error("Failed to get ordered queue for semaphore notification")
		return nil, err
	}
	return queue, nil
}

// notifyWaiters enqueues the next N workflows who are waiting for the semaphore to the workqueue,
// where N is the availability of the semaphore. If semaphore is out of capacity, this does nothing.
func (s *databaseSemaphore) notifyWaiters() {
	limit := s.getLimit()
	// We don't need to run a transaction here, if we get it wrong it'll right itself
	holders, err := s.getCurrentHolders()
	if err != nil {
		s.log.WithError(err).Error("Failed to notify waiters")
		return
	}
	holdCount := len(holders)

	pending, err := s.queueOrdered(s.info.session)
	if err != nil {
		return
	}
	triggerCount := min(limit-holdCount, len(pending))
	s.log.WithFields(log.Fields{
		"holdCount":    holdCount,
		"triggerCount": triggerCount,
		"pendingCount": len(pending),
	}).Debug("Notifying waiters for semaphore")
	for idx := 0; idx < triggerCount; idx++ {
		item := pending[idx]
		if item.Controller != s.info.config.controllerName {
			continue
		}
		key := workflowKey(item.Key)
		s.log.WithFields(log.Fields{"key": item.Key, "workflowKey": key}).Debug("Enqueueing workflow for semaphore notification")
		s.nextWorkflow(key)
	}
}

// addToQueue adds the holderkey into priority queue that maintains the priority order to acquire the lock.
func (s *databaseSemaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) error {
	// Doesn't need a transaction, as no-one else should be inserting exactly this record ever
	var states []stateRecord
	err := s.info.session.SQL().
		Select(stateKeyField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.longDbKey()}).
		And(db.Cond{stateKeyField: holderKey}).
		And(db.Cond{stateControllerField: s.info.config.controllerName}).
		All(&states)
	if err != nil {
		return err
	}
	if len(states) > 0 {
		return nil
	}
	record := &stateRecord{
		Name:       s.longDbKey(),
		Key:        holderKey,
		Controller: s.info.config.controllerName,
		Held:       false,
		Priority:   priority,
		Time:       creationTime,
	}
	_, err = s.info.session.Collection(s.info.config.stateTable).Insert(record)
	return err
}

func (s *databaseSemaphore) removeFromQueue(holderKey string) error {
	_, err := s.info.session.SQL().
		DeleteFrom(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.longDbKey()}).
		And(db.Cond{stateKeyField: holderKey}).
		And(db.Cond{stateHeldField: false}).
		Exec()

	return err
}

func (s *databaseSemaphore) checkAcquire(holderKey string, tx *transaction) (bool, bool, string) {
	if holderKey == "" {
		s.log.WithFields(log.Fields{
			"result":       false,
			"already_held": false,
			"message":      "bug: attempt to check semaphore with empty holder key",
		}).Info("CheckAcquire failed")
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit()
	holders, err := s.currentHoldersSession(*tx.db)
	if err != nil {
		s.log.WithFields(log.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"error":        err.Error(),
		}).Info("CheckAcquire failed")
		return false, false, err.Error()
	}
	if slices.Contains(holders, holderKey) {
		s.log.WithFields(log.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": true,
		}).Info("CheckAcquire - already held")
		return false, true, ""
	}
	waitingMsg := fmt.Sprintf("Waiting for %s lock (%s). Lock status: %d/%d", s.name, s.longDbKey(), len(holders), limit)

	if len(holders) >= limit {
		s.log.WithFields(log.Fields{
			"key":             holderKey,
			"result":          false,
			"already_held":    false,
			"message":         waitingMsg,
			"current_holders": len(holders),
			"limit":           limit,
		}).Info("CheckAcquire - limit exceeded")
		return false, false, waitingMsg
	}
	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	// Only live controllers are considered
	queue, err := s.queueOrdered(*tx.db)
	if err != nil {
		s.log.WithFields(log.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"error":        err.Error(),
		}).Info("CheckAcquire failed")
		return false, false, err.Error()
	}
	if len(queue) == 0 {
		s.log.WithFields(log.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
		}).Info("CheckAcquire - empty queue")
		return false, false, ""
	}
	if queue[0].Controller != s.info.config.controllerName {
		s.log.WithFields(log.Fields{
			"key":                holderKey,
			"result":             false,
			"already_held":       false,
			"message":            waitingMsg,
			"queue_controller":   queue[0].Controller,
			"current_controller": s.info.config.controllerName,
		}).Info("CheckAcquire - different controller")
		return false, false, waitingMsg
	}
	if !isSameWorkflowNodeKeys(holderKey, queue[0].Key) {
		// Enqueue the queue[0] workflow if lock is available
		if len(holders) < limit {
			s.nextWorkflow(queue[0].Key)
		}
		s.log.WithFields(log.Fields{
			"key":          holderKey,
			"result":       false,
			"already_held": false,
			"message":      waitingMsg,
			"queue_key":    queue[0].Key,
		}).Info("CheckAcquire - not first in queue")
		return false, false, waitingMsg
	}
	s.log.WithFields(log.Fields{
		"key":          holderKey,
		"result":       true,
		"already_held": false,
	}).Info("CheckAcquire - can acquire")
	return true, false, ""
}

func (s *databaseSemaphore) acquire(holderKey string, tx *transaction) bool {
	limit := s.getLimit()
	existing, err := s.currentHoldersSession(*tx.db)
	if err != nil {
		s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
		return false
	}
	if len(existing) < limit {
		var pending []stateRecord
		err := (*tx.db).SQL().
			Select(stateKeyField).
			From(s.info.config.stateTable).
			Where(db.Cond{stateNameField: s.longDbKey()}).
			And(db.Cond{stateKeyField: holderKey}).
			And(db.Cond{stateControllerField: s.info.config.controllerName}).
			And(db.Cond{stateHeldField: false}).
			All(&pending)
		if err != nil {
			s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
			return false
		}
		if len(pending) > 0 {
			_, err := (*tx.db).SQL().Update(s.info.config.stateTable).
				Set(stateHeldField, true).
				Where(db.Cond{stateNameField: s.longDbKey()}).
				And(db.Cond{stateKeyField: holderKey}).
				And(db.Cond{stateControllerField: s.info.config.controllerName}).
				And(db.Cond{stateHeldField: false}).
				Exec()
			if err != nil {
				s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
				return false
			}
		} else {
			record := &stateRecord{
				Name:       s.longDbKey(),
				Key:        holderKey,
				Controller: s.info.config.controllerName,
				Held:       true,
			}
			_, err := (*tx.db).Collection(s.info.config.stateTable).Insert(record)
			if err != nil {
				s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
				return false
			}
		}
		s.log.WithFields(log.Fields{
			"key":    holderKey,
			"result": true,
		}).Info("Acquire succeeded")
		return true
	}
	s.log.WithFields(log.Fields{
		"key":             holderKey,
		"result":          false,
		"reason":          "limit exceeded",
		"current_holders": len(existing),
		"limit":           limit,
	}).Info("Acquire failed")
	return false
}

func (s *databaseSemaphore) tryAcquire(holderKey string, tx *transaction) (bool, string) {
	acq, already, msg := s.checkAcquire(holderKey, tx)
	if already {
		s.log.WithFields(log.Fields{
			"key":     holderKey,
			"result":  true,
			"message": msg,
		}).Info("tryAcquire - already held")
		return true, msg
	}
	if !acq {
		s.log.WithFields(log.Fields{
			"key":     holderKey,
			"result":  false,
			"message": msg,
		}).Info("tryAcquire - cannot acquire")
		return false, msg
	}
	if s.acquire(holderKey, tx) {
		s.log.WithFields(log.Fields{
			"key":    holderKey,
			"result": true,
		}).Info("tryAcquire succeeded")
		s.notifyWaiters()
		return true, ""
	}
	s.log.WithFields(log.Fields{
		"key":     holderKey,
		"result":  false,
		"message": msg,
	}).Info("tryAcquire failed")
	return false, msg
}

func (s *databaseSemaphore) expireLocks() {
	since := time.Now().Add(-s.info.config.inactiveControllerTimeout)
	subquery := s.info.session.SQL().
		Select(controllerNameField).
		From(s.info.config.controllerTable).
		And(db.Cond{controllerTimeField + " <=": since})

	// Delete locks from inactive controllers
	result, err := s.info.session.SQL().DeleteFrom(s.info.config.lockTable).
		Where(db.Cond{lockControllerField + " IN": subquery}).
		Exec()
	if err != nil {
		s.log.WithError(err).Error("Failed to expire locks")
	} else if rowsAffected, err := result.RowsAffected(); err == nil && rowsAffected > 0 {
		s.log.WithField("rowsAffected", rowsAffected).Info("Expired locks")
	}
}

func (s *databaseSemaphore) probeWaiting() {
	s.notifyWaiters()
	s.expireLocks()
}
