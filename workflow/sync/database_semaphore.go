package sync

import (
	"fmt"
	"slices"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/upper/db/v4"
)

type dbConfig struct {
	limitTable                string
	stateTable                string
	controllerTable           string
	controllerName            string
	inactiveControllerTimeout time.Duration
}

type databaseSemaphore struct {
	name         string
	limitGetter  limitProvider
	dbKey        string
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
	Mutex      bool      `db:"mutex"`
	Held       bool      `db:"held"`
	Priority   int32     `db:"priority"` // higher number = higher priority in queue
	Time       time.Time `db:"time"`     // timestamp of creation or last update
}

type controllerHealthRecord struct {
	Controller string    `db:"controller"` // controller where the workflow is running
	Time       time.Time `db:"time"`       // timestamp of creation or last update
}

const (
	limitNameField = "name"
	limitSizeField = "sizelimit"

	stateNameField       = "name"
	stateKeyField        = "workflowkey"
	stateControllerField = "controller"
	stateMutexField      = "mutex"
	stateHeldField       = "held"
	statePriorityField   = "priority"
	stateTimeField       = "time"

	controllerNameField = "controller"
	controllerTimeField = "time"
)

var _ semaphore = &databaseSemaphore{}

func newDatabaseSemaphore(name string, dbKey string, nextWorkflow NextWorkflow, info dbInfo, syncLimitCacheTTL time.Duration) *databaseSemaphore {
	sm := &databaseSemaphore{
		name:         name,
		dbKey:        dbKey,
		limitGetter:  nil,
		nextWorkflow: nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeSemaphore,
			"name":     name,
		}),
		info:    info,
		isMutex: false,
	}
	sm.limitGetter = newCachedLimit(sm.getLimitFromDB, syncLimitCacheTTL)
	return sm
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
		Where(db.Cond{limitNameField: s.dbKey}).
		One(limit)
	if err != nil {
		s.log.WithField("key", s.dbKey).WithError(err).Error("Failed to get limit")
		return 0, err
	}
	s.log.WithFields(log.Fields{
		"limit": limit.SizeLimit,
		"key":   s.dbKey,
	}).Debug("Current limit")
	return limit.SizeLimit, nil
}

// getLimit returns the semaphore limit. If isMutex this always returns 1.
// Otherwise queries the database for the limit.
func (s *databaseSemaphore) getLimit() int {
	log.WithFields(log.Fields{
		"dbKey": s.dbKey,
	}).Infof("getLimit")
	limit, _, err := s.limitGetter.get(s.dbKey)
	if err != nil {
		s.log.WithError(err).Errorf("Failed to get limit for semaphore %s", s.name)
		return 0
	}
	return limit
}

func (s *databaseSemaphore) currentState(session db.Session, held bool) []string {
	var states []stateRecord
	err := session.SQL().
		Select(stateKeyField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateHeldField: held}).
		And(db.Cond{stateNameField: s.dbKey}).
		And(db.Cond{stateMutexField: s.isMutex}).
		All(&states)
	if err != nil {
		s.log.WithField("held", held).WithError(err).Error("Failed to get current state")
	}
	keys := make([]string, len(states))
	for i := range states {
		keys[i] = states[i].Key
	}
	return keys
}

func (s *databaseSemaphore) getCurrentPending() []string {
	return s.currentState(s.info.session, false)
}

func (s *databaseSemaphore) getCurrentHolders() []string {
	return s.currentHoldersSession(s.info.session)
}

func (s *databaseSemaphore) currentHoldersSession(session db.Session) []string {
	return s.currentState(session, true)
}

func (s *databaseSemaphore) release(key string) bool {
	_, err := s.info.session.SQL().
		DeleteFrom(s.info.config.stateTable).
		Where(db.Cond{stateHeldField: true}).
		And(db.Cond{stateNameField: s.dbKey}).
		And(db.Cond{stateKeyField: key}).
		And(db.Cond{stateControllerField: s.info.config.controllerName}).
		And(db.Cond{stateMutexField: s.isMutex}).
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
	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	// Only live controllers are considered
	since := time.Now().Add(-s.info.config.inactiveControllerTimeout)
	var queue []stateRecord
	err := session.SQL().
		Select(stateKeyField, stateControllerField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.dbKey}).
		And(db.Cond{stateHeldField: false}).
		And(db.Cond{stateMutexField: s.isMutex}).
		And(db.Cond{
			"controller IN": session.SQL().
				Select(controllerNameField).
				From(s.info.config.controllerTable).
				And(db.Cond{controllerTimeField + " >": since}),
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
	holdCount := len(s.getCurrentHolders())

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
func (s *databaseSemaphore) addToQueue(holderKey string, priority int32, creationTime time.Time, tx *transaction) {
	var states []stateRecord
	err := (*tx.db).SQL().
		Select(stateKeyField).
		From(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.dbKey}).
		And(db.Cond{stateKeyField: holderKey}).
		And(db.Cond{stateControllerField: s.info.config.controllerName}).
		And(db.Cond{stateMutexField: s.isMutex}).
		All(&states)
	if err != nil {
		s.log.WithField("key", holderKey).WithError(err).Error("Failed to add to queue")
		return
	}
	if len(states) > 0 {
		return
	}
	_, err = (*tx.db).Collection(s.info.config.stateTable).
		Insert(&stateRecord{
			Name:       s.dbKey,
			Key:        holderKey,
			Controller: s.info.config.controllerName,
			Held:       false,
			Mutex:      s.isMutex,
			Priority:   priority,
			Time:       creationTime,
		})
	switch err {
	case nil:
		s.log.WithField("key", holderKey).Debug("Added into queue")
	default:
		s.log.WithField("key", holderKey).WithError(err).Error("Failed to add to queue")
	}
}

func (s *databaseSemaphore) removeFromQueue(holderKey string) {
	_, err := s.info.session.SQL().
		DeleteFrom(s.info.config.stateTable).
		Where(db.Cond{stateNameField: s.dbKey}).
		And(db.Cond{stateKeyField: holderKey}).
		And(db.Cond{stateHeldField: false}).
		Exec()
	switch err {
	case nil:
		s.log.WithField("key", holderKey).Debug("Removed from queue")
	default:
		s.log.WithField("key", holderKey).WithError(err).Error("Failed to remove from queue")
	}
}

func (s *databaseSemaphore) acquire(holderKey string, tx *transaction) bool {
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit()
	existing := s.currentHoldersSession(*tx.db)
	if len(existing) < limit {
		var pending []stateRecord
		err := (*tx.db).SQL().
			Select(stateKeyField).
			From(s.info.config.stateTable).
			Where(db.Cond{stateNameField: s.dbKey}).
			And(db.Cond{stateKeyField: holderKey}).
			And(db.Cond{stateControllerField: s.info.config.controllerName}).
			And(db.Cond{stateMutexField: s.isMutex}).
			And(db.Cond{stateHeldField: false}).
			All(&pending)
		if err != nil {
			s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
			return false
		}
		if len(pending) > 0 {
			// Update the existing row in this transaction - removeFromQueue will
			// fail later
			_, err := (*tx.db).SQL().Update(s.info.config.stateTable).
				Set(stateHeldField, true).
				Where(db.Cond{stateNameField: s.dbKey}).
				And(db.Cond{stateKeyField: holderKey}).
				And(db.Cond{stateControllerField: s.info.config.controllerName}).
				And(db.Cond{stateMutexField: s.isMutex}).
				And(db.Cond{stateHeldField: false}).
				Exec()
			if err != nil {
				s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
				return false
			}
		} else { // insert
			_, err := (*tx.db).Collection(s.info.config.stateTable).
				Insert(&stateRecord{
					Name:       s.dbKey,
					Key:        holderKey,
					Controller: s.info.config.controllerName,
					Mutex:      s.isMutex,
					Held:       true,
				})
			if err != nil {
				s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
				return false
			}
		}
		return true
	}
	return false
}

// checkAcquire examines if tryAcquire would succeed
// returns
//
//	true, false if we would be able to take the lock
//	false, true if we already have the lock
//	false, false if the lock is not acquirable
//	string return is a user facing message when not acquirable
func (s *databaseSemaphore) checkAcquire(holderKey string, tx *transaction) (bool, bool, string) {
	if holderKey == "" {
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit()
	holders := s.currentHoldersSession(*tx.db)
	if slices.Contains(holders, holderKey) {
		return false, true, ""
	}
	waitingMsg := fmt.Sprintf("Waiting for %s lock (%s). Lock status: %d/%d", s.name, s.dbKey, len(holders), limit)

	if len(holders) >= limit {
		return false, false, waitingMsg
	}
	// Check whether requested holdkey is in front of priority queue.
	// If it is in front position, it will allow to acquire lock.
	// If it is not a front key, it needs to wait for its turn.
	// Only live controllers are considered
	queue, err := s.queueOrdered(*tx.db)
	if err != nil {
		s.log.WithFields(log.Fields{
			"key":       holderKey,
			"semaphore": s.dbKey,
		}).WithError(err).Error("Failed to check if lock can be acquired")
		return false, false, err.Error()
	}
	if len(queue) == 0 {
		return false, false, ""
	}
	if queue[0].Controller != s.info.config.controllerName {
		return false, false, ""
	}
	if !isSameWorkflowNodeKeys(holderKey, queue[0].Key) {
		// Enqueue the queue[0] workflow if lock is available
		if len(holders) < limit {
			s.nextWorkflow(queue[0].Key)
		}
		return false, false, waitingMsg
	}
	return true, false, ""
}

func (s *databaseSemaphore) tryAcquire(holderKey string, tx *transaction) (bool, string) {
	acq, already, msg := s.checkAcquire(holderKey, tx)
	if already {
		return true, msg
	}
	if !acq {
		return false, msg
	}
	if s.acquire(holderKey, tx) {
		s.log.WithField("key", holderKey).Debug("Successfully acquired lock")
		s.notifyWaiters()
		return true, ""
	}
	return false, msg
}

func (s *databaseSemaphore) probeWaiting() {
	s.notifyWaiters()
}
