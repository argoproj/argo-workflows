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
	name              string
	limit             int
	limitTimestamp    time.Time
	syncLimitCacheTTL time.Duration
	dbKey             string
	nextWorkflow      NextWorkflow
	log               *log.Entry
	info              dbInfo
	isMutex           bool
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
	return &databaseSemaphore{
		name:              name,
		dbKey:             dbKey,
		limit:             0,
		limitTimestamp:    time.Time{}, // cause a refresh first call to getLimit
		syncLimitCacheTTL: syncLimitCacheTTL,
		nextWorkflow:      nextWorkflow,
		log: log.WithFields(log.Fields{
			"lockType": lockTypeSemaphore,
			"name":     name,
		}),
		info:    info,
		isMutex: false,
	}
}

func (s *databaseSemaphore) getName() string {
	return s.name
}

func (s *databaseSemaphore) updateLimitFromDB() error {
	// Update the limit from the database
	limit := &limitRecord{}
	err := s.info.session.SQL().
		Select(limitSizeField).
		From(s.info.config.limitTable).
		Where(db.Cond{limitNameField: s.dbKey}).
		One(limit)
	if err != nil {
		s.log.WithField("key", s.dbKey).WithError(err).Error("Failed to get limit")
		return err
	}
	s.log.WithFields(log.Fields{
		"limit": limit.SizeLimit,
		"key":   s.dbKey,
	}).Debug("Current limit")
	s.resetLimitTimestamp()
	s.limit = limit.SizeLimit
	return nil
}

// getLimit returns the semaphore limit. If isMutex this always returns 1.
// Otherwise queries the database for the limit.
func (s *databaseSemaphore) getLimit() int {
	log.WithFields(log.Fields{
		"dbKey":             s.dbKey,
		"isMutex":           s.isMutex,
		"limitTimestamp":    s.getLimitTimestamp(),
		"syncLimitCacheTTL": s.syncLimitCacheTTL,
		"remaining":         nowFn().Sub(s.getLimitTimestamp()),
	}).Infof("getLimit")
	if !s.isMutex && nowFn().Sub(s.getLimitTimestamp()) >= s.syncLimitCacheTTL {
		if s.updateLimitFromDB() != nil {
			return 0
		}
	}
	return s.limit
}

func (s *databaseSemaphore) getLimitTimestamp() time.Time {
	return s.limitTimestamp
}

func (s *databaseSemaphore) resetLimitTimestamp() {
	s.limitTimestamp = nowFn()
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
func (s *databaseSemaphore) addToQueue(holderKey string, priority int32, creationTime time.Time) {
	err := s.info.session.Tx(func(sess db.Session) error {
		var states []stateRecord
		err := sess.SQL().
			Select(stateKeyField).
			From(s.info.config.stateTable).
			Where(db.Cond{stateNameField: s.dbKey}).
			And(db.Cond{stateKeyField: holderKey}).
			And(db.Cond{stateControllerField: s.info.config.controllerName}).
			And(db.Cond{stateMutexField: s.isMutex}).
			All(&states)
		if err != nil {
			return err
		}
		if len(states) > 0 {
			return nil
		}
		_, err = sess.Collection(s.info.config.stateTable).
			Insert(&stateRecord{
				Name:       s.dbKey,
				Key:        holderKey,
				Controller: s.info.config.controllerName,
				Held:       false,
				Mutex:      s.isMutex,
				Priority:   priority,
				Time:       creationTime,
			})
		return err
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

func (s *databaseSemaphore) acquire(holderKey string) bool {
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit()
	result := false
	err := s.info.session.Tx(func(sess db.Session) error {
		existing := s.currentHoldersSession(sess)
		if len(existing) < limit {
			var pending []stateRecord
			err := sess.SQL().
				Select(stateKeyField).
				From(s.info.config.stateTable).
				Where(db.Cond{stateNameField: s.dbKey}).
				And(db.Cond{stateKeyField: holderKey}).
				And(db.Cond{stateControllerField: s.info.config.controllerName}).
				And(db.Cond{stateMutexField: s.isMutex}).
				And(db.Cond{stateHeldField: false}).
				All(&pending)
			if err != nil {
				return err
			}
			if len(pending) > 0 {
				// Update the existing row in this transaction - removeFromQueue will
				// fail later
				_, err := sess.SQL().Update(s.info.config.stateTable).
					Set(stateHeldField, true).
					Where(db.Cond{stateNameField: s.dbKey}).
					And(db.Cond{stateKeyField: holderKey}).
					And(db.Cond{stateControllerField: s.info.config.controllerName}).
					And(db.Cond{stateMutexField: s.isMutex}).
					And(db.Cond{stateHeldField: false}).
					Exec()
				if err != nil {
					return err
				}
			} else { // insert
				_, err := sess.Collection(s.info.config.stateTable).
					Insert(&stateRecord{
						Name:       s.dbKey,
						Key:        holderKey,
						Controller: s.info.config.controllerName,
						Mutex:      s.isMutex,
						Held:       true,
					})
				if err != nil {
					return err
				}
			}
			result = true
			return nil
		}
		return nil
	})
	if err != nil {
		s.log.WithField("key", holderKey).WithError(err).Error("Failed to acquire lock")
	}
	return result
}

// checkAcquire examines if tryAcquire would succeed
// returns
//
//	true, false if we would be able to take the lock
//	false, true if we already have the lock
//	false, false if the lock is not acquirable
//	string return is a user facing message when not acquirable
func (s *databaseSemaphore) checkAcquire(holderKey string) (bool, bool, string) {
	if holderKey == "" {
		return false, false, "bug: attempt to check semaphore with empty holder key"
	}
	// Limit changes are eventually consistent, not inside the tx
	limit := s.getLimit()
	acquirable := false
	already := false
	msg := ""
	err := s.info.session.Tx(func(sess db.Session) error {
		holders := s.currentHoldersSession(sess)
		if slices.Contains(holders, holderKey) {
			already = true
			return nil
		}

		waitingMsg := fmt.Sprintf("Waiting for %s lock (%s). Lock status: %d/%d", s.name, s.dbKey, len(holders), limit)

		if len(holders) >= limit {
			msg = waitingMsg
			return nil
		}
		// Check whether requested holdkey is in front of priority queue.
		// If it is in front position, it will allow to acquire lock.
		// If it is not a front key, it needs to wait for its turn.
		// Only live controllers are considered
		queue, err := s.queueOrdered(sess)
		if err != nil {
			return err
		}
		if len(queue) == 0 {
			return nil
		}
		if queue[0].Controller != s.info.config.controllerName {
			return nil
		}
		if !isSameWorkflowNodeKeys(holderKey, queue[0].Key) {
			// Enqueue the queue[0] workflow if lock is available
			if len(holders) < limit {
				s.nextWorkflow(queue[0].Key)
			}
			msg = waitingMsg
			return nil
		}

		acquirable = true
		return nil
	})
	if err != nil {
		s.log.WithFields(log.Fields{
			"key":       holderKey,
			"semaphore": s.dbKey,
		}).WithError(err).Error("Failed to check if lock can be acquired")
	}

	return acquirable, already, msg
}

func (s *databaseSemaphore) resize(n int) bool {
	s.log.WithField("requestedSize", n).Debug("Database semaphores don't support resizing")
	return false
}

func (s *databaseSemaphore) tryAcquire(holderKey string) (bool, string) {
	acq, already, msg := s.checkAcquire(holderKey)
	if already {
		return true, msg
	}
	if !acq {
		return false, msg
	}
	if s.acquire(holderKey) {
		s.log.WithField("key", holderKey).Debug("Successfully acquired lock")
		s.notifyWaiters()
		return true, ""
	}
	return false, msg
}

func (s *databaseSemaphore) probeWaiting() {
	s.notifyWaiters()
}
