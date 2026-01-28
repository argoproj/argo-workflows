package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/upper/db/v4"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	syncdb "github.com/argoproj/argo-workflows/v3/util/sync/db"
)

type (
	NextWorkflow   func(string)
	GetSyncLimit   func(context.Context, string) (int, error)
	WorkflowExists func(string) bool
)

type Manager struct {
	syncLockMap       map[string]semaphore
	lock              *sync.RWMutex
	nextWorkflow      NextWorkflow
	getSyncLimit      GetSyncLimit
	syncLimitCacheTTL time.Duration
	workflowExists    WorkflowExists
	dbInfo            syncdb.DBInfo
	queries           syncdb.SyncQueries
	log               logging.Logger
}

type lockTypeName string

const (
	lockTypeSemaphore lockTypeName = "semaphore"
	lockTypeMutex     lockTypeName = "mutex"
)

func NewLockManager(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, config *config.SyncConfig, getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, workflowExists WorkflowExists) *Manager {
	return createLockManager(ctx, syncdb.DBSessionFromConfig(ctx, kubectlConfig, namespace, config), config, getSyncLimit, nextWorkflow, workflowExists)
}

func createLockManager(ctx context.Context, dbSession db.Session, config *config.SyncConfig, getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, workflowExists WorkflowExists) *Manager {
	syncLimitCacheTTL := time.Duration(0)
	if config != nil && config.SemaphoreLimitCacheSeconds != nil {
		syncLimitCacheTTL = time.Duration(*config.SemaphoreLimitCacheSeconds) * time.Second
	}
	ctx, log := logging.RequireLoggerFromContext(ctx).WithField("component", "lock_manager").InContext(ctx)

	log.WithField("syncLimitCacheTTL", syncLimitCacheTTL).Info(ctx, "Sync manager ttl")
	dbInfo := syncdb.DBInfo{
		Session: dbSession,
		Config:  syncdb.DBConfigFromConfig(config),
	}
	sm := &Manager{
		syncLockMap:       make(map[string]semaphore),
		lock:              &sync.RWMutex{},
		nextWorkflow:      nextWorkflow,
		getSyncLimit:      getSyncLimit,
		syncLimitCacheTTL: syncLimitCacheTTL,
		workflowExists:    workflowExists,
		dbInfo:            dbInfo,
		queries:           syncdb.NewSyncQueries(dbSession, dbInfo.Config),
		log:               log,
	}
	log.WithField("dbConfigured", sm.dbInfo.Session != nil).Info(ctx, "Sync manager initialized")
	sm.dbInfo.Migrate(ctx)

	if sm.dbInfo.Session != nil {
		sm.backgroundNotifier(ctx, config.PollSeconds)
		sm.dbControllerHeartbeat(ctx, config.HeartbeatSeconds)
	}
	return sm
}

func (sm *Manager) getWorkflowKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("holderkey is empty")
	}
	items := strings.Split(key, "/")
	if len(items) < 2 {
		return "", fmt.Errorf("invalid holderkey format")
	}
	return fmt.Sprintf("%s/%s", items[0], items[1]), nil
}

func (sm *Manager) CheckWorkflowExistence(ctx context.Context) {
	defer runtimeutil.HandleCrashWithContext(ctx, runtimeutil.PanicHandlers...)

	sm.lock.RLock()
	defer sm.lock.RUnlock()

	sm.log.Debug(ctx, "Check the workflow existence")
	for _, lock := range sm.syncLockMap {
		holders, err := lock.getCurrentHolders(ctx)
		if err != nil {
			sm.log.WithError(err).Error(ctx, "failed to get current lock holders")
			continue
		}
		pending, err := lock.getCurrentPending(ctx)
		if err != nil {
			sm.log.WithError(err).Error(ctx, "failed to get current lock pending")
			continue
		}
		keys := append(holders, pending...)
		for _, holderKeys := range keys {
			wfKey, err := sm.getWorkflowKey(holderKeys)
			if err != nil {
				continue
			}
			if !sm.workflowExists(wfKey) {
				lock.release(ctx, holderKeys)
				if err := lock.removeFromQueue(ctx, holderKeys); err != nil {
					sm.log.WithField("holderKeys", holderKeys).WithError(err).Warn(ctx, "failed to remove from queue")
				}
			}
		}
	}
}

func getUpgradedKey(wf *wfv1.Workflow, key string, level SyncLevelType) string {
	if wfv1.CheckHolderKeyVersion(key) == wfv1.HoldingNameV1 {
		if level == WorkflowLevel {
			return getHolderKey(wf, "")
		}
		return getHolderKey(wf, key)
	}
	return key
}

type SyncLevelType int

const (
	WorkflowLevel SyncLevelType = 1
	TemplateLevel SyncLevelType = 2
	ErrorLevel    SyncLevelType = 3
)

// HoldingNameV1 keys can be of the form
// x where x is a workflow name
// unfortunately this doesn't differentiate between workflow level keys
// and template level keys. So upgrading is a bit tricky here.

// given a legacy holding name x, namespace y and workflow name z.
// in the case of a workflow level
// if x != z
// upgradedKey := y/z
// elseif x == z
// upgradedKey := y/z
// in the case of a template level
// if x != z
// upgradedKey := y/z/x
// elif x == z
// upgradedKey := y/z/x

// there is a possibility that
// a synchronization exists both at the template level
// and at the workflow level -> impossible to upgrade correctly
// due to ambiguity. Currently we just assume workflow level.
func getWorkflowSyncLevelByName(ctx context.Context, wf *wfv1.Workflow, lockName string) (SyncLevelType, error) {
	if wf.Spec.Synchronization != nil {
		syncItems, err := allSyncItems(wf.Spec.Synchronization)
		if err != nil {
			return ErrorLevel, err
		}
		for _, syncItem := range syncItems {
			syncLockName, err := syncItem.lockName(wf.Namespace)
			if err != nil {
				return ErrorLevel, err
			}
			if lockName == syncLockName.String(ctx) {
				return WorkflowLevel, nil
			}
		}
	}

	var lastErr error
	for _, template := range wf.Spec.Templates {
		if template.Synchronization != nil {
			syncItems, err := allSyncItems(template.Synchronization)
			if err != nil {
				return ErrorLevel, err
			}
			for _, syncItem := range syncItems {
				syncLockName, err := syncItem.lockName(wf.Namespace)
				if err != nil {
					lastErr = err
					continue
				}
				if lockName == syncLockName.String(ctx) {
					return TemplateLevel, nil
				}
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("was unable to determine level for %s", lockName)
	}
	return ErrorLevel, lastErr
}

func (sm *Manager) Initialize(ctx context.Context, wfs []wfv1.Workflow) {
	for _, wf := range wfs {
		if wf.Status.Synchronization == nil {
			continue
		}

		if wf.Status.Synchronization.Semaphore != nil {
			for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
				semaphore := sm.syncLockMap[holding.Semaphore]
				if semaphore == nil {
					semaphore, err := sm.initializeSemaphore(ctx, holding.Semaphore)
					if err != nil {
						sm.log.WithField("semaphore", holding.Semaphore).WithError(err).Warn(ctx, "cannot initialize semaphore")
						continue
					}
					sm.syncLockMap[holding.Semaphore] = semaphore
				}

				for _, holders := range holding.Holders {
					level, err := getWorkflowSyncLevelByName(ctx, &wf, holding.Semaphore)
					if err != nil {
						sm.log.WithField("semaphore", holding.Semaphore).WithError(err).Warn(ctx, "cannot obtain lock level")
						continue
					}
					key := getUpgradedKey(&wf, holders, level)
					tx := &transaction{db: &sm.dbInfo.Session}
					if semaphore != nil && semaphore.acquire(ctx, key, tx) {
						sm.log.WithFields(logging.Fields{"key": key, "semaphore": holding.Semaphore}).Info(ctx, "Lock acquired")
					}
				}

			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {
				mutex := sm.syncLockMap[holding.Mutex]
				if mutex == nil {
					mutex, err := sm.initializeMutex(ctx, holding.Mutex)
					if err != nil {
						sm.log.WithField("mutex", holding.Mutex).WithError(err).Warn(ctx, "cannot initialize mutex")
						continue
					}
					if holding.Holder != "" {
						level, err := getWorkflowSyncLevelByName(ctx, &wf, holding.Mutex)
						if err != nil {
							sm.log.WithField("mutex", holding.Mutex).WithError(err).Warn(ctx, "cannot obtain lock level")
							continue
						}
						key := getUpgradedKey(&wf, holding.Holder, level)
						tx := &transaction{db: &sm.dbInfo.Session}
						mutex.acquire(ctx, key, tx)
					}
					sm.syncLockMap[holding.Mutex] = mutex
				}
			}
		}
	}
	sm.log.Info(ctx, "Sync manager initialized successfully")
}

// TryAcquire tries to acquire the lock from semaphore.
// It returns status of acquiring a lock , status of Workflow status updated, waiting message if lock is not available, the failed lock, and any error encountered
func (sm *Manager) TryAcquire(ctx context.Context, wf *wfv1.Workflow, nodeName string, syncLockRef *wfv1.Synchronization) (bool, bool, string, string, error) {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	if syncLockRef == nil {
		return false, false, "", "", fmt.Errorf("cannot acquire lock from nil Synchronization")
	}

	failedLockName := ""
	syncItems, err := allSyncItems(syncLockRef)
	if err != nil {
		return false, false, "", failedLockName, fmt.Errorf("requested configuration is invalid: %w", err)
	}
	holderKey := getHolderKey(wf, nodeName)

	lockKeys := make([]string, len(syncItems))
	for i, syncItem := range syncItems {
		syncLockName, err := syncItem.lockName(wf.Namespace)
		if err != nil {
			return false, false, "", failedLockName, fmt.Errorf("requested configuration is invalid: %w", err)
		}
		sm.log.WithField("syncLockName", syncLockName).Info(ctx, "TryAcquire")
		lockKeys[i] = syncLockName.String(ctx)
	}

	if ok, msg, failedLockName, err := sm.prepAcquire(ctx, wf, holderKey, syncItems, lockKeys); !ok {
		return false, false, msg, failedLockName, err
	}

	needDB, err := needDBSession(ctx, lockKeys)
	if err != nil {
		return false, false, "", failedLockName, fmt.Errorf("couldn't decode locks for session: %w", err)
	}
	if needDB && sm.dbInfo.Session == nil {
		return false, false, "", failedLockName, fmt.Errorf("synchronization database session is not available")
	}
	if needDB {
		var updated bool
		var already bool
		var msg string
		var failedLockName string
		var lastErr error
		for retryCounter := range 5 {
			err := sm.dbInfo.Session.TxContext(ctx, func(sess db.Session) error {
				sm.log.WithFields(logging.Fields{
					"holderKey": holderKey,
					"attempt":   retryCounter + 1,
				}).Info(ctx, "TryAcquire - starting transaction")
				var err error
				tx := &transaction{db: &sess}
				already, updated, msg, failedLockName, err = sm.tryAcquireImpl(ctx, wf, tx, holderKey, failedLockName, syncItems, lockKeys)
				sm.log.WithFields(logging.Fields{
					"holderKey": holderKey,
					"attempt":   retryCounter + 1,
				}).Info(ctx, "TryAcquire - transaction completed")
				return err
			}, &sql.TxOptions{
				Isolation: sql.LevelSerializable,
				ReadOnly:  false,
			})
			if err == nil {
				return already, updated, msg, failedLockName, nil
			}
			lastErr = err
			// Check if this is a serialization error
			if strings.Contains(err.Error(), "serialization") ||
				strings.Contains(err.Error(), "dependencies") ||
				strings.Contains(err.Error(), "deadlock") ||
				strings.Contains(err.Error(), "rollback") {
				sm.log.WithFields(logging.Fields{
					"holderKey": holderKey,
					"attempt":   retryCounter + 1,
					"error":     err,
				}).Info(ctx, "TryAcquire - serialization conflict, retrying")
				continue
			} else {
				sm.log.WithFields(logging.Fields{
					"holderKey": holderKey,
					"attempt":   retryCounter + 1,
					"error":     err,
				}).Info(ctx, "TryAcquire - tx failed")
			}
			// For other errors, return immediately
			return false, false, "", failedLockName, err
		}
		return false, false, "", failedLockName, fmt.Errorf("failed after %d retries: %w", 5, lastErr)
	}
	return sm.tryAcquireImpl(ctx, wf, nil, holderKey, failedLockName, syncItems, lockKeys)
}

func (sm *Manager) prepAcquire(ctx context.Context, wf *wfv1.Workflow, holderKey string, syncItems []*syncItem, lockKeys []string) (bool, string, string, error) {
	for i, lockKey := range lockKeys {
		lock, found := sm.syncLockMap[lockKey]
		if !found {
			var err error
			switch syncItems[i].getType() {
			case wfv1.SynchronizationTypeSemaphore:
				lock, err = sm.initializeSemaphore(ctx, lockKey)
			case wfv1.SynchronizationTypeMutex:
				lock, err = sm.initializeMutex(ctx, lockKey)
			default:
				return false, "bug: unknown synchronization type in prepAcquire", lockKey, fmt.Errorf("unknown Synchronization Type")
			}
			if err != nil {
				return false, "failed to initialize lock", lockKey, err
			}
			sm.syncLockMap[lockKey] = lock
		}

		var priority int32
		if wf.Spec.Priority != nil {
			priority = *wf.Spec.Priority
		} else {
			priority = 0
		}
		creationTime := wf.CreationTimestamp
		ensureInit(wf, syncItems[i].getType())
		if err := lock.addToQueue(ctx, holderKey, priority, creationTime.Time); err != nil {
			return false, fmt.Sprintf("Failed to add to queue: %v", err), lockKey, err
		}
	}
	return true, "", "", nil
}

func (sm *Manager) tryAcquireImpl(ctx context.Context, wf *wfv1.Workflow, tx *transaction, holderKey string, failedLockName string, syncItems []*syncItem, lockKeys []string) (bool, bool, string, string, error) {
	defer sm.unlockAll(ctx, lockKeys)
	allAcquirable := true
	msg := ""
	for _, lockKey := range lockKeys {
		lock, found := sm.syncLockMap[lockKey]
		if !found {
			return false, false, "", failedLockName, fmt.Errorf("bug: lock not found: %s", lockKey)
		}
		if lock.lock(ctx) {
			acquired, already, newMsg := lock.checkAcquire(ctx, holderKey, tx)
			if !acquired && !already {
				allAcquirable = false
				if msg == "" {
					msg = newMsg
				}
				if failedLockName == "" {
					failedLockName = lockKey
				}
			}
		} else {
			allAcquirable = false
			msg = "failed to lock()"
			if failedLockName == "" {
				failedLockName = lockKey
			}
		}
	}

	switch {
	case allAcquirable:
		updated := false
		for i, lockKey := range lockKeys {
			lock := sm.syncLockMap[lockKey]
			acquired, msg := lock.tryAcquire(ctx, holderKey, tx)
			if !acquired {
				return false, false, "", failedLockName, fmt.Errorf("bug: failed to acquire something that should have been checked: %s", msg)
			}
			currentHolders, err := sm.getCurrentLockHolders(ctx, lockKey)
			if err != nil {
				return false, false, "", failedLockName, fmt.Errorf("failed to get current lock holders: %w", err)
			}
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockAcquired(holderKey, lockKey, currentHolders) {
				updated = true
			}
		}
		return true, updated, msg, failedLockName, nil
	default: // Not all acquirable
		updated := false
		for i, lockKey := range lockKeys {
			currentHolders, err := sm.getCurrentLockHolders(ctx, lockKey)
			if err != nil {
				return false, false, "", failedLockName, fmt.Errorf("failed to get current lock holders: %w", err)
			}
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockWaiting(holderKey, lockKey, currentHolders) {
				updated = true
			}
		}
		return false, updated, msg, failedLockName, nil
	}
}

func (sm *Manager) unlockAll(ctx context.Context, lockKeys []string) {
	for _, lockKey := range lockKeys {
		lock := sm.syncLockMap[lockKey]
		lock.unlock(ctx)
	}
}

func (sm *Manager) Release(ctx context.Context, wf *wfv1.Workflow, nodeName string, syncRef *wfv1.Synchronization) {
	if syncRef == nil {
		return
	}

	sm.lock.RLock()
	defer sm.lock.RUnlock()

	holderKey := getHolderKey(wf, nodeName)
	sm.log.WithField("holderKey", holderKey).Info(ctx, "Release")
	// Ignoring error here is as good as it's going to be, we shouldn't get here as we should
	// should never have acquired anything if this errored
	syncItems, _ := allSyncItems(syncRef)

	for _, syncItem := range syncItems {
		lockName, err := syncItem.lockName(wf.Namespace)
		if err != nil {
			return
		}
		if syncLockHolder, ok := sm.syncLockMap[lockName.String(ctx)]; ok {
			syncLockHolder.release(ctx, holderKey)
			if err := syncLockHolder.removeFromQueue(ctx, holderKey); err != nil {
				sm.log.WithField("holderKey", holderKey).WithError(err).Warn(ctx, "Error removing from queue")
			}
			lockKey := lockName
			if wf.Status.Synchronization != nil {
				wf.Status.Synchronization.GetStatus(syncItem.getType()).LockReleased(holderKey, lockKey.String(ctx))
			}
		}
	}
}

func (sm *Manager) ReleaseAll(ctx context.Context, wf *wfv1.Workflow) bool {
	sm.lock.RLock()
	defer sm.lock.RUnlock()

	if wf.Status.Synchronization == nil {
		return true
	}

	if wf.Status.Synchronization.Semaphore != nil {
		for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
			syncLockHolder := sm.syncLockMap[holding.Semaphore]
			if syncLockHolder == nil {
				continue
			}

			for _, holderKey := range holding.Holders {
				syncLockHolder.release(ctx, holderKey)
				wf.Status.Synchronization.Semaphore.LockReleased(holderKey, holding.Semaphore)
				sm.log.WithFields(logging.Fields{"holderKey": holderKey, "semaphore": holding.Semaphore}).Info(ctx, "Lock released")
			}
		}

		// Remove the pending Workflow level semaphore keys
		for _, waiting := range wf.Status.Synchronization.Semaphore.Waiting {
			syncLockHolder := sm.syncLockMap[waiting.Semaphore]
			if syncLockHolder == nil {
				continue
			}
			key := getHolderKey(wf, "")
			if err := syncLockHolder.removeFromQueue(ctx, key); err != nil {
				sm.log.WithField("key", key).WithError(err).Warn(ctx, "Error removing from queue")
			}
		}
		wf.Status.Synchronization.Semaphore = nil
	}

	if wf.Status.Synchronization.Mutex != nil {
		h := make([]wfv1.MutexHolding, len(wf.Status.Synchronization.Mutex.Holding))
		copy(h, wf.Status.Synchronization.Mutex.Holding)
		for _, holding := range h {
			syncLockHolder := sm.syncLockMap[holding.Mutex]
			if syncLockHolder == nil {
				continue
			}

			syncLockHolder.release(ctx, holding.Holder)
			wf.Status.Synchronization.Mutex.LockReleased(holding.Holder, holding.Mutex)
			sm.log.WithFields(logging.Fields{"holderKey": holding.Holder, "mutex": holding.Mutex}).Info(ctx, "Lock released")
		}

		// Remove the pending Workflow level mutex keys
		for _, waiting := range wf.Status.Synchronization.Mutex.Waiting {
			syncLockHolder := sm.syncLockMap[waiting.Mutex]
			if syncLockHolder == nil {
				continue
			}
			key := getHolderKey(wf, "")
			if err := syncLockHolder.removeFromQueue(ctx, key); err != nil {
				sm.log.WithField("key", key).WithError(err).Warn(ctx, "Error removing from queue")
			}
		}
		wf.Status.Synchronization.Mutex = nil
	}

	for _, node := range wf.Status.Nodes {
		if node.SynchronizationStatus != nil && node.SynchronizationStatus.Waiting != "" {
			lock, ok := sm.syncLockMap[node.SynchronizationStatus.Waiting]
			if ok {
				if err := lock.removeFromQueue(ctx, getHolderKey(wf, node.ID)); err != nil {
					sm.log.WithField("key", getHolderKey(wf, node.ID)).WithError(err).Warn(ctx, "Error removing from queue")
				}
			}
			node.SynchronizationStatus = nil
			wf.Status.Nodes.Set(ctx, node.ID, node)
		}
	}

	wf.Status.Synchronization = nil
	return true
}

func ensureInit(wf *wfv1.Workflow, lockType wfv1.SynchronizationType) {
	if wf.Status.Synchronization == nil {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{}
	}
	if lockType == wfv1.SynchronizationTypeSemaphore && wf.Status.Synchronization.Semaphore == nil {
		wf.Status.Synchronization.Semaphore = &wfv1.SemaphoreStatus{}
	}
	if lockType == wfv1.SynchronizationTypeMutex && wf.Status.Synchronization.Mutex == nil {
		wf.Status.Synchronization.Mutex = &wfv1.MutexStatus{}
	}
}

func getHolderKey(wf *wfv1.Workflow, nodeName string) string {
	if wf == nil {
		return ""
	}
	key := fmt.Sprintf("%s/%s", wf.Namespace, wf.Name)
	if nodeName != "" {
		key = fmt.Sprintf("%s/%s", key, nodeName)
	}
	return key
}

func (sm *Manager) getCurrentLockHolders(ctx context.Context, lock string) ([]string, error) {
	if concurrency, ok := sm.syncLockMap[lock]; ok {
		return concurrency.getCurrentHolders(ctx)
	}
	return nil, nil
}

func (sm *Manager) initializeSemaphore(ctx context.Context, semaphoreName string) (semaphore, error) {
	lock, err := DecodeLockName(ctx, semaphoreName)
	if err != nil {
		return nil, err
	}
	switch lock.Kind {
	case lockKindConfigMap:
		return newInternalSemaphore(ctx, semaphoreName, sm.nextWorkflow, sm.getSyncLimit, sm.syncLimitCacheTTL)
	case lockKindDatabase:
		if sm.dbInfo.Session == nil {
			return nil, fmt.Errorf("database session is not available for semaphore %s", semaphoreName)
		}
		return newDatabaseSemaphore(ctx, semaphoreName, lock.dbKey(), sm.nextWorkflow, sm.dbInfo, sm.syncLimitCacheTTL)
	default:
		return nil, fmt.Errorf("invalid lock kind %s when initializing semaphore", lock.Kind)
	}
}

func (sm *Manager) initializeMutex(ctx context.Context, mutexName string) (semaphore, error) {
	lock, err := DecodeLockName(ctx, mutexName)
	if err != nil {
		return nil, err
	}
	switch lock.Kind {
	case lockKindMutex:
		return newInternalMutex(mutexName, sm.nextWorkflow), nil
	case lockKindDatabase:
		if sm.dbInfo.Session == nil {
			return nil, fmt.Errorf("database session is not available for mutex %s", mutexName)
		}
		return newDatabaseMutex(mutexName, lock.dbKey(), sm.nextWorkflow, sm.dbInfo), nil
	default:
		return nil, fmt.Errorf("invalid lock kind %s when initializing mutex", lock.Kind)
	}
}

func (sm *Manager) backgroundNotifier(ctx context.Context, period *int) {
	sm.log.WithField("pollInterval", syncdb.SecondsToDurationWithDefault(period, syncdb.DefaultDBHeartbeatSeconds)).
		Info(ctx, "Starting background notification for sync locks")
	go wait.UntilWithContext(ctx, func(_ context.Context) {
		sm.lock.Lock()
		for _, lock := range sm.syncLockMap {
			lock.probeWaiting(ctx)
		}
		sm.lock.Unlock()
	},
		syncdb.SecondsToDurationWithDefault(period, syncdb.DefaultDBPollSeconds),
	)
}

// dbControllerHeartbeat does periodic deadmans switch updates to the controller state
func (sm *Manager) dbControllerHeartbeat(ctx context.Context, period *int) {
	// This doesn't need be be transactional, if someone else has the same controller name as us
	// you've got much worse problems
	// Failure here is not critical, so we don't check errors, we may already be in the table
	ll := db.LC().Level()
	db.LC().SetLevel(db.LogLevelError)
	_ = sm.queries.InsertControllerHealth(ctx, &syncdb.ControllerHealthRecord{
		Controller: sm.dbInfo.Config.ControllerName,
		Time:       time.Now(),
	})
	db.LC().SetLevel(ll)

	sm.dbControllerUpdate(ctx)
	go wait.UntilWithContext(ctx, func(_ context.Context) { sm.dbControllerUpdate(ctx) },
		syncdb.SecondsToDurationWithDefault(period, syncdb.DefaultDBHeartbeatSeconds))
}

func (sm *Manager) dbControllerUpdate(ctx context.Context) {
	err := sm.queries.UpdateControllerTimestamp(ctx, sm.dbInfo.Config.ControllerName, time.Now())
	if err != nil {
		sm.log.WithError(err).Error(ctx, "Failed to update sync controller timestamp")
	}
}
