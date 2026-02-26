package db

import (
	"context"
	"time"

	"github.com/upper/db/v4"

	"github.com/argoproj/argo-workflows/v4/util/sqldb"
)

// Record types for database operations
type LimitRecord struct {
	Name      string `db:"name"`
	SizeLimit int    `db:"sizelimit"`
}

type StateRecord struct {
	Name       string    `db:"name"`        // semaphore name identifier
	Key        string    `db:"workflowkey"` // workflow key holding or waiting for the lock of the form <namespace>/<name>
	Controller string    `db:"controller"`  // controller where the workflow is running
	Held       bool      `db:"held"`
	Priority   int32     `db:"priority"` // higher number = higher priority in queue
	Time       time.Time `db:"time"`     // timestamp of creation or last update
}

type ControllerHealthRecord struct {
	Controller string    `db:"controller"` // controller where the workflow is running
	Time       time.Time `db:"time"`       // timestamp of creation or last update
}

type LockRecord struct {
	Name       string    `db:"name"`       // semaphore name identifier
	Controller string    `db:"controller"` // controller where the workflow is running
	Time       time.Time `db:"time"`       // timestamp of creation
}

// Field name constants
const (
	LimitNameField = "name"
	LimitSizeField = "sizelimit"

	StateNameField       = "name"
	StateKeyField        = "workflowkey"
	StateControllerField = "controller"
	StateHeldField       = "held"
	StatePriorityField   = "priority"
	StateTimeField       = "time"

	ControllerNameField = "controller"
	ControllerTimeField = "time"

	LockNameField       = "name"
	LockControllerField = "controller"
)

type SyncQueries interface {
	CreateSemaphoreLimit(ctx context.Context, name string, sizeLimit int) error
	UpdateSemaphoreLimit(ctx context.Context, name string, sizeLimit int) error
	DeleteSemaphoreLimit(ctx context.Context, name string) error
	GetSemaphoreLimit(ctx context.Context, dbKey string) (*LimitRecord, error)

	GetCurrentState(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string, held bool) ([]StateRecord, error)
	GetCurrentHolders(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string) ([]StateRecord, error)
	GetCurrentPending(ctx context.Context, semaphoreName string) ([]StateRecord, error)
	GetOrderedQueue(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string, inactiveTimeout time.Duration) ([]StateRecord, error)
	AddToQueue(ctx context.Context, record *StateRecord) error
	RemoveFromQueue(ctx context.Context, semaphoreName, holderKey string) error
	CheckQueueExists(ctx context.Context, semaphoreName, holderKey, controllerName string) ([]StateRecord, error)
	UpdateStateToHeld(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) error
	InsertHeldState(ctx context.Context, session *sqldb.SessionProxy, record *StateRecord) error
	GetPendingInQueue(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) ([]StateRecord, error)
	ReleaseHeld(ctx context.Context, semaphoreName, key, controllerName string) error

	GetExistingLocks(ctx context.Context, lockName, controllerName string) ([]LockRecord, error)
	InsertLock(ctx context.Context, record *LockRecord) error
	DeleteLock(ctx context.Context, lockName string) error
	ExpireInactiveLocks(ctx context.Context, inactiveTimeout time.Duration) (int64, error)

	InsertControllerHealth(ctx context.Context, record *ControllerHealthRecord) error
	UpdateControllerTimestamp(ctx context.Context, controllerName string, timestamp time.Time) error

	GetPendingInQueueWithSession(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) ([]StateRecord, error)
	UpdateStateToHeldWithSession(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) error
	InsertHeldStateWithSession(ctx context.Context, session *sqldb.SessionProxy, record *StateRecord) error
}

var _ SyncQueries = &syncQueries{}

// syncQueries holds all SQL query operations for the sync package
type syncQueries struct {
	config       Config
	sessionProxy *sqldb.SessionProxy
}

// NewSyncQueries creates a new syncQueries instance
func NewSyncQueries(sessionProxy *sqldb.SessionProxy, config Config) SyncQueries {
	return &syncQueries{
		config:       config,
		sessionProxy: sessionProxy,
	}
}

// Limit operations
func (q *syncQueries) CreateSemaphoreLimit(ctx context.Context, name string, sizeLimit int) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.LimitTable).Insert(&LimitRecord{
			Name:      name,
			SizeLimit: sizeLimit,
		})
		return err
	})
}

func (q *syncQueries) UpdateSemaphoreLimit(ctx context.Context, name string, sizeLimit int) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		resp, err := s.SQL().Update(q.config.LimitTable).
			Set(LimitSizeField, sizeLimit).
			Where(db.Cond{LimitNameField: name}).
			Exec()

		if err != nil {
			return err
		}

		affectedRows, err := resp.RowsAffected()
		if err != nil {
			return err
		}

		if affectedRows == 0 {
			return db.ErrNoMoreRows
		}

		return nil
	})
}

func (q *syncQueries) DeleteSemaphoreLimit(ctx context.Context, name string) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.SQL().DeleteFrom(q.config.LimitTable).
			Where(db.Cond{LimitNameField: name}).
			Exec()
		return err
	})
}

func (q *syncQueries) GetSemaphoreLimit(ctx context.Context, name string) (*LimitRecord, error) {
	var limit *LimitRecord
	err := q.sessionProxy.With(ctx, func(s db.Session) error {
		limit = &LimitRecord{}
		err := s.SQL().
			Select(LimitSizeField).
			From(q.config.LimitTable).
			Where(db.Cond{LimitNameField: name}).
			One(limit)
		return err
	})
	return limit, err
}

// State operations
func (q *syncQueries) GetCurrentState(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string, held bool) ([]StateRecord, error) {
	var states []StateRecord
	err := session.With(ctx, func(s db.Session) error {
		err := s.SQL().
			Select(StateKeyField).
			From(q.config.StateTable).
			Where(db.Cond{StateHeldField: held}).
			And(db.Cond{StateNameField: semaphoreName}).
			All(&states)
		return err
	})
	return states, err
}

func (q *syncQueries) GetCurrentHolders(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string) ([]StateRecord, error) {
	return q.GetCurrentState(ctx, session, semaphoreName, true)
}

func (q *syncQueries) GetCurrentPending(ctx context.Context, semaphoreName string) ([]StateRecord, error) {
	return q.GetCurrentState(ctx, q.sessionProxy, semaphoreName, false)
}

func (q *syncQueries) GetOrderedQueue(ctx context.Context, session *sqldb.SessionProxy, semaphoreName string, inactiveTimeout time.Duration) ([]StateRecord, error) {
	var queue []StateRecord
	err := session.With(ctx, func(s db.Session) error {
		since := time.Now().Add(-inactiveTimeout)
		subquery := s.SQL().
			Select(ControllerNameField).
			From(q.config.ControllerTable).
			And(db.Cond{ControllerTimeField + " >": since})

		err := s.SQL().
			Select(StateKeyField, StateControllerField).
			From(q.config.StateTable).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateHeldField: false}).
			And(db.Cond{
				"controller IN": subquery,
			}).
			OrderBy(StatePriorityField+" DESC", StateTimeField+" ASC").
			All(&queue)
		return err
	})
	return queue, err
}

func (q *syncQueries) AddToQueue(ctx context.Context, record *StateRecord) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.StateTable).Insert(record)
		return err
	})
}

func (q *syncQueries) RemoveFromQueue(ctx context.Context, semaphoreName, holderKey string) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.SQL().
			DeleteFrom(q.config.StateTable).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateHeldField: false}).
			Exec()
		return err
	})
}

func (q *syncQueries) CheckQueueExists(ctx context.Context, semaphoreName, holderKey, controllerName string) ([]StateRecord, error) {
	var states []StateRecord
	err := q.sessionProxy.With(ctx, func(s db.Session) error {
		err := s.SQL().
			Select(StateKeyField).
			From(q.config.StateTable).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateControllerField: controllerName}).
			All(&states)
		return err
	})
	return states, err
}

func (q *syncQueries) UpdateStateToHeld(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) error {
	return session.With(ctx, func(s db.Session) error {
		_, err := s.SQL().Update(q.config.StateTable).
			Set(StateHeldField, true).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateControllerField: controllerName}).
			And(db.Cond{StateHeldField: false}).
			Exec()
		return err
	})
}

func (q *syncQueries) InsertHeldState(ctx context.Context, session *sqldb.SessionProxy, record *StateRecord) error {
	return session.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.StateTable).Insert(record)
		return err
	})
}

func (q *syncQueries) GetPendingInQueue(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) ([]StateRecord, error) {
	var pending []StateRecord
	err := session.With(ctx, func(s db.Session) error {
		err := s.SQL().
			Select(StateKeyField).
			From(q.config.StateTable).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateControllerField: controllerName}).
			And(db.Cond{StateHeldField: false}).
			All(&pending)
		return err
	})
	return pending, err
}

func (q *syncQueries) ReleaseHeld(ctx context.Context, semaphoreName, key, controllerName string) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.SQL().
			DeleteFrom(q.config.StateTable).
			Where(db.Cond{StateHeldField: true}).
			And(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: key}).
			And(db.Cond{StateControllerField: controllerName}).
			Exec()
		return err
	})
}

// Lock operations
func (q *syncQueries) GetExistingLocks(ctx context.Context, lockName, controllerName string) ([]LockRecord, error) {
	var existingLocks []LockRecord
	err := q.sessionProxy.With(ctx, func(s db.Session) error {
		err := s.SQL().
			Select(LockNameField).
			From(q.config.LockTable).
			Where(db.Cond{LockNameField: lockName}).
			And(db.Cond{LockControllerField: controllerName}).
			All(&existingLocks)
		return err
	})
	return existingLocks, err
}

func (q *syncQueries) InsertLock(ctx context.Context, record *LockRecord) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.LockTable).Insert(record)
		return err
	})
}

func (q *syncQueries) DeleteLock(ctx context.Context, lockName string) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.SQL().
			DeleteFrom(q.config.LockTable).
			Where(db.Cond{LockNameField: lockName}).
			Exec()
		return err
	})
}

func (q *syncQueries) ExpireInactiveLocks(ctx context.Context, inactiveTimeout time.Duration) (int64, error) {
	var rowsAffected int64
	err := q.sessionProxy.With(ctx, func(s db.Session) error {
		since := time.Now().Add(-inactiveTimeout)
		subquery := s.SQL().
			Select(ControllerNameField).
			From(q.config.ControllerTable).
			And(db.Cond{ControllerTimeField + " <=": since})

		result, err := s.SQL().DeleteFrom(q.config.LockTable).
			Where(db.Cond{LockControllerField + " IN": subquery}).
			Exec()
		if err != nil {
			return err
		}
		rowsAffected, err = result.RowsAffected()
		return err
	})
	return rowsAffected, err
}

// Controller operations
func (q *syncQueries) InsertControllerHealth(ctx context.Context, record *ControllerHealthRecord) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.ControllerTable).Insert(record)
		return err
	})
}

func (q *syncQueries) UpdateControllerTimestamp(ctx context.Context, controllerName string, timestamp time.Time) error {
	return q.sessionProxy.With(ctx, func(s db.Session) error {
		_, err := s.SQL().Update(q.config.ControllerTable).
			Set(ControllerTimeField, timestamp).
			Where(db.Cond{ControllerNameField: controllerName}).
			Exec()
		return err
	})
}

// Transaction-based operations for acquire/release operations
func (q *syncQueries) GetPendingInQueueWithSession(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) ([]StateRecord, error) {
	var pending []StateRecord
	err := session.With(ctx, func(s db.Session) error {
		err := s.SQL().
			Select(StateKeyField).
			From(q.config.StateTable).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateControllerField: controllerName}).
			And(db.Cond{StateHeldField: false}).
			All(&pending)
		return err
	})
	return pending, err
}

func (q *syncQueries) UpdateStateToHeldWithSession(ctx context.Context, session *sqldb.SessionProxy, semaphoreName, holderKey, controllerName string) error {
	return session.With(ctx, func(s db.Session) error {
		_, err := s.SQL().Update(q.config.StateTable).
			Set(StateHeldField, true).
			Where(db.Cond{StateNameField: semaphoreName}).
			And(db.Cond{StateKeyField: holderKey}).
			And(db.Cond{StateControllerField: controllerName}).
			And(db.Cond{StateHeldField: false}).
			Exec()
		return err
	})
}

func (q *syncQueries) InsertHeldStateWithSession(ctx context.Context, session *sqldb.SessionProxy, record *StateRecord) error {
	return session.With(ctx, func(s db.Session) error {
		_, err := s.Collection(q.config.StateTable).Insert(record)
		return err
	})
}
