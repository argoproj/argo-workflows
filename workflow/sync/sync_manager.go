package sync

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/upper/db/v4"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/sqldb"
	syncdb "github.com/argoproj/argo-workflows/v4/util/sync/db"
	wfmetrics "github.com/argoproj/argo-workflows/v4/workflow/metrics"
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
	dbInfo            syncdb.Info
	queries           syncdb.SyncQueries
	log               logging.Logger
	metrics           syncMetrics
}

func (sm *Manager) WithMetrics(m *wfmetrics.Metrics) *Manager {
	sm.metrics = m
	return sm
}

type lockTypeName string

const (
	lockTypeSemaphore lockTypeName = "semaphore"
	lockTypeMutex     lockTypeName = "mutex"
)

// NewLockManager creates a new lock manager
func NewLockManager(ctx context.Context, kubectlConfig kubernetes.Interface, namespace string, config *config.SyncConfig, getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, workflowExists WorkflowExists, ensureDBConnection bool) (*Manager, error) {
	var sessionProxy *sqldb.SessionProxy
	var err error
	if config != nil && ensureDBConnection {
		sessionProxy, err = sqldb.NewSessionProxy(ctx, sqldb.SessionProxyConfig{KubectlConfig: kubectlConfig, Namespace: namespace, DBConfig: config.DBConfig})
		if err != nil {
			return nil, err
		}
	}
	return createLockManager(ctx, sessionProxy, config, getSyncLimit, nextWorkflow, workflowExists), nil
}

func createLockManager(ctx context.Context, sessionProxy *sqldb.SessionProxy, config *config.SyncConfig, getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, workflowExists WorkflowExists) *Manager {
	syncLimitCacheTTL := time.Duration(0)
	if config != nil && config.SemaphoreLimitCacheSeconds != nil {
		syncLimitCacheTTL = time.Duration(*config.SemaphoreLimitCacheSeconds) * time.Second
	}
	ctx, log := logging.RequireLoggerFromContext(ctx).WithField("component", "lock_manager").InContext(ctx)

	log.WithField("syncLimitCacheTTL", syncLimitCacheTTL).Info(ctx, "Sync manager ttl")
	dbInfo := syncdb.Info{
		SessionProxy: sessionProxy,
		Config:       syncdb.ConfigFromConfig(config),
	}
	sm := &Manager{
		syncLockMap:       make(map[string]semaphore),
		lock:              &sync.RWMutex{},
		nextWorkflow:      nextWorkflow,
		getSyncLimit:      getSyncLimit,
		syncLimitCacheTTL: syncLimitCacheTTL,
		workflowExists:    workflowExists,
		dbInfo:            dbInfo,
		queries:           syncdb.NewSyncQueries(sessionProxy, dbInfo.Config),
		log:               log,
	}
	log.WithField("dbConfigured", sm.dbInfo.SessionProxy != nil).Info(ctx, "Sync manager initialized")
	sm.dbInfo.Migrate(ctx)

	if sm.dbInfo.SessionProxy != nil {
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

	sm.lock.Lock()
	defer sm.lock.Unlock()

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
		keys := slices.Concat(holders, pending)
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

func getUpgradedKey(wf *wfv1.Workflow, key string, level LevelType) string {
	if wfv1.CheckHolderKeyVersion(key) == wfv1.HoldingNameV1 {
		if level == WorkflowLevel {
			return getHolderKey(wf, "")
		}
		return getHolderKey(wf, key)
	}
	return key
}

// upgradeHolderKey resolves a holder key recorded in a Workflow's
// synchronization status into the key form the in-memory lock expects.
//
// V2 keys are self-describing - they already encode whether the hold is at the
// workflow level (ns/wfname) or the template level (ns/wfname/nodeID) - so they
// are returned verbatim and no spec lookup is needed. Only legacy V1 keys are
// ambiguous and require getWorkflowSyncLevelByName to determine the level. This
// matters for workflowTemplateRef workflows, whose wf.Spec is empty: a V2 key
// can be re-established without ever resolving a level from the spec.
func upgradeHolderKey(ctx context.Context, wf *wfv1.Workflow, holderKey, lockName string) (string, error) {
	if wfv1.CheckHolderKeyVersion(holderKey) != wfv1.HoldingNameV1 {
		return holderKey, nil
	}
	level, err := getWorkflowSyncLevelByName(ctx, wf, lockName)
	if err != nil {
		return "", err
	}
	return getUpgradedKey(wf, holderKey, level), nil
}

type LevelType int

const (
	WorkflowLevel LevelType = 1
	TemplateLevel LevelType = 2
	ErrorLevel    LevelType = 3
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
func getWorkflowSyncLevelByName(ctx context.Context, wf *wfv1.Workflow, lockName string) (LevelType, error) {
	// For workflowTemplateRef workflows wf.Spec.Synchronization and
	// wf.Spec.Templates are empty; the rendered spec lives in
	// wf.Status.StoredWorkflowSpec. Inspect both so the level can be resolved
	// regardless of where the synchronization block was declared.
	syncBlocks := []*wfv1.Synchronization{wf.Spec.Synchronization}
	templates := wf.Spec.Templates
	if wf.Status.StoredWorkflowSpec != nil {
		syncBlocks = append(syncBlocks, wf.Status.StoredWorkflowSpec.Synchronization)
		// slices.Concat allocates a fresh backing array; a plain append could
		// write into wf.Spec.Templates' spare capacity and corrupt the caller's
		// slice (which aliases the workflow in wfs).
		templates = slices.Concat(wf.Spec.Templates, wf.Status.StoredWorkflowSpec.Templates)
	}

	for _, sync := range syncBlocks {
		if sync == nil {
			continue
		}
		syncItems, err := allSyncItems(sync)
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
	for _, template := range templates {
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

// initFailureFatal reports whether a failure to (re)establish a lock at startup
// is unrecoverable and must fail closed (crashloop). Only two cases qualify:
//   - the lock name is undecodable, so there is no key to poison under and no way
//     to prove the workflow's spec re-acquires the same lock; and
//   - the lock is database-backed but no database session is configured, so
//     nothing can back the lock.
//
// Everything else - a transient ConfigMap/DB read failure, a limit fetch
// returning 0 - is recoverable: the name decodes, so the lock can be poisoned
// (the poison key matches what a racer would compute) without halting the whole
// controller.
func (sm *Manager) initFailureFatal(ctx context.Context, lockName string) bool {
	decoded, err := DecodeLockName(ctx, lockName)
	if err != nil {
		return true
	}
	return decoded.getKind() == lockKindDatabase && sm.dbInfo.SessionProxy == nil
}

// poison installs a poisoned lock, refusing all acquires until the next restart.
func (sm *Manager) poison(ctx context.Context, lockName, reason string) {
	sm.log.WithField("lock", lockName).WithField("reason", reason).Warn(ctx, "poisoning lock")
	sm.syncLockMap[lockName] = newPoisonedLock(lockName, reason)
}

// reestablishHolder re-establishes a single recorded holder of lockName in the
// in-memory lock map. lockType is "semaphore" or "mutex" (for logging) and
// initLock builds the backing lock when it is not yet present.
//
// It always reads the current lock from the map (never a stale local), so a lock
// poisoned by a previous holder stays poisoned and is not acquired on an orphaned
// object. A returned error is fatal (see initFailureFatal). An init failure or an
// unresolvable holder key poisons the lock. A non-empty staleReason means the
// hold could not be verified against its backing store (database-backed locks
// only); the caller fails the workflow, whose teardown releases its locks.
//
// Poisoning, not leaving absent, is required for the init-failure case: a
// ConfigMap-backed semaphore keeps its holders only in memory, so if we left the
// lock absent, prepAcquire would later rebuild it with zero holders once the
// backend recovered and let a racer acquire the slot this holder still owns. The
// poison is lock-scoped and clears on the next controller restart.
func (sm *Manager) reestablishHolder(ctx context.Context, wf *wfv1.Workflow, lockType, lockName, holder string, initLock func(context.Context, string) (semaphore, error)) (staleReason string, fatalErr error) {
	if sm.syncLockMap[lockName] == nil {
		lock, err := initLock(ctx, lockName)
		if err != nil {
			if sm.initFailureFatal(ctx, lockName) {
				// Undecodable name or a database hold with no session: we cannot
				// poison to protect the recorded hold, so halt for an operator
				// rather than risk a silent double-acquire.
				sm.log.WithField(lockType, lockName).WithError(err).Error(ctx, "cannot initialize lock, failing closed")
				return "", fmt.Errorf("cannot re-establish %s %q held by workflow %s/%s at startup: %w", lockType, lockName, wf.Namespace, wf.Name, err)
			}
			// Recoverable (e.g. transient ConfigMap unavailability) but the name
			// decodes, so poison protects the recorded hold without crashlooping.
			// Leaving the lock absent would be unsound: an in-memory semaphore
			// rebuilt later would have zero holders and let a racer double-acquire.
			sm.poison(ctx, lockName, fmt.Sprintf("controller could not initialize lock at startup: %v", err))
			return "", nil
		}
		sm.syncLockMap[lockName] = lock
	}

	if holder == "" {
		return "", nil
	}

	key, err := upgradeHolderKey(ctx, wf, holder, lockName)
	if err != nil {
		sm.poison(ctx, lockName, fmt.Sprintf("controller could not re-establish recorded holder %q at startup: %v", holder, err))
		return "", nil
	}

	// Re-read from the map: a previous holder of this same lock may have poisoned
	// it, in which case reacquire is a no-op and we must not resurrect it.
	lock := sm.syncLockMap[lockName]
	// For in-memory locks reacquire force-registers the hold ignoring the limit,
	// so the recorded hold is always represented: dropping it would let a racer
	// double-acquire a semaphore whose holders exceed a lowered limit, and
	// poisoning would block the whole shared lock over a routine limit change.
	// For database-backed locks the database is the single source of truth and
	// reacquire only asserts the hold is still recorded there; if it is not, the
	// workflow's recorded hold is stale and the workflow is failed rather than
	// left to run on a hold the database no longer backs.
	if err := lock.reacquire(ctx, key, sm.dbInfo.SessionProxy); err != nil {
		sm.log.WithFields(logging.Fields{"key": key, lockType: lockName}).WithError(err).Warn(ctx, "could not re-establish recorded holder, failing the workflow")
		return fmt.Sprintf("could not re-establish %s %q at controller startup: %v", lockType, lockName, err), nil
	}
	sm.log.WithFields(logging.Fields{"key": key, lockType: lockName}).Info(ctx, "re-established recorded holder")
	return "", nil
}

// StaleHold records a workflow whose recorded hold on a database-backed lock
// could not be verified against the database during Initialize. The database
// is the single source of truth for such locks, so the workflow is running on
// a hold the database no longer backs (e.g. it was expired while the
// controller was down and may since have been acquired by another holder).
// The controller fails these workflows; their teardown releases any locks
// they still hold.
type StaleHold struct {
	WF     *wfv1.Workflow
	Reason string
}

// Initialize re-establishes, in the in-memory lock map, the holds that Running
// workflows record in their status.
//
// It fails closed only when a holder is genuinely unrecoverable (see
// initFailureFatal): an undecodable lock name, or a database-backed hold with no
// database session. Those return an error the controller treats as fatal,
// because we can neither poison the lock nor prove the spec re-acquires it, so
// continuing risks a silent double-acquire.
//
// Recoverable failures never crashloop: a lock that cannot be built (transient
// ConfigMap/DB read) or whose holder key is unresolvable is poisoned (lock-scoped,
// clears on restart). A database-backed hold that the database no longer records
// is returned as a StaleHold (at most one per workflow) for the controller to
// fail the workflow.
func (sm *Manager) Initialize(ctx context.Context, wfs []wfv1.Workflow) ([]StaleHold, error) {
	// Hold the lock for the whole pass: a DB-backed Manager starts its
	// backgroundNotifier goroutine in createLockManager (before initManagers
	// calls Initialize), and that goroutine iterates syncLockMap under sm.lock.
	sm.lock.Lock()
	defer sm.lock.Unlock()

	var staleHolds []StaleHold
	for i := range wfs {
		wf := &wfs[i]
		if wf.Status.Synchronization == nil {
			continue
		}

		// Record only the first stale hold per workflow (failing it once is
		// enough), but keep re-establishing its remaining holds: they are real
		// until the failed workflow's teardown releases them, and dropping one
		// from the in-memory map would let a racer double-acquire it.
		stale := func(reason string) {
			// neat way to prevent double counting workflows, this works because we iterate workflow by workflow
			// we only need to ask was the last stale workflow the same as this one as a result.
			if len(staleHolds) == 0 || staleHolds[len(staleHolds)-1].WF != wf {
				staleHolds = append(staleHolds, StaleHold{WF: wf, Reason: reason})
			}
		}

		if wf.Status.Synchronization.Semaphore != nil {
			for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
				for _, holder := range holding.Holders {
					reason, err := sm.reestablishHolder(ctx, wf, "semaphore", holding.Semaphore, holder, func(ctx context.Context, name string) (semaphore, error) {
						return sm.initializeSemaphore(ctx, name)
					})
					if err != nil {
						return nil, err
					}
					if reason != "" {
						stale(reason)
					}
				}
			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {
				reason, err := sm.reestablishHolder(ctx, wf, "mutex", holding.Mutex, holding.Holder, func(ctx context.Context, name string) (semaphore, error) {
					return sm.initializeMutex(ctx, name)
				})
				if err != nil {
					return nil, err
				}
				if reason != "" {
					stale(reason)
				}
			}
		}
	}
	sm.log.Info(ctx, "Sync manager initialized successfully")
	return staleHolds, nil
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
		syncLockName, lockNameErr := syncItem.lockName(wf.Namespace)
		if lockNameErr != nil {
			return false, false, "", failedLockName, fmt.Errorf("requested configuration is invalid: %w", lockNameErr)
		}
		sm.log.WithField("syncLockName", syncLockName).Info(ctx, "TryAcquire")
		lockKeys[i] = syncLockName.String(ctx)
	}

	if ok, msg, prepLockName, prepErr := sm.prepAcquire(ctx, wf, holderKey, syncItems, lockKeys); !ok {
		return false, false, msg, prepLockName, prepErr
	}

	needDB, err := needDBSession(ctx, lockKeys)
	if err != nil {
		return false, false, "", failedLockName, fmt.Errorf("couldn't decode locks for session: %w", err)
	}
	if needDB && sm.dbInfo.SessionProxy == nil {
		return false, false, "", failedLockName, fmt.Errorf("synchronization database session is not available")
	}
	if needDB {
		var updated bool
		var already bool
		var msg string
		var newly []*acquiredLock
		// Backoff bounds: sm.lock is held for the whole loop, so cap each sleep
		// modestly. Jitter prevents a fleet of replicas from retrying in lockstep
		// after a shared conflict burst.
		backoff := wait.Backoff{
			Steps:    5,
			Duration: 10 * time.Millisecond,
			Factor:   2.0,
			Jitter:   0.5,
			Cap:      600 * time.Millisecond,
		}
		attempt := 0
		err = retry.OnError(backoff, isRetryableSyncError, func() error {
			attempt++
			sm.log.WithFields(logging.Fields{
				"holderKey": holderKey,
				"attempt":   attempt,
			}).Info(ctx, "TryAcquire - starting transaction")
			txErr := sm.dbInfo.SessionProxy.TxWith(ctx, func(sp *sqldb.SessionProxy) error {
				var implErr error
				already, updated, msg, failedLockName, newly, implErr = sm.tryAcquireImpl(ctx, wf, sp, holderKey, failedLockName, syncItems, lockKeys)
				return implErr
			}, &sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: false})
			if txErr != nil {
				sm.log.WithFields(logging.Fields{
					"holderKey": holderKey,
					"attempt":   attempt,
					"error":     txErr,
					"retryable": isRetryableSyncError(txErr),
				}).Info(ctx, "TryAcquire - transaction failed")
			}
			return txErr
		})
		if err != nil {
			return false, false, "", failedLockName, err
		}
		sm.recordAcquisitions(ctx, wf, newly)
		return already, updated, msg, failedLockName, nil
	}
	already, updated, msg, failedLockName, newly, err := sm.tryAcquireImpl(ctx, wf, nil, holderKey, failedLockName, syncItems, lockKeys)
	if err == nil {
		sm.recordAcquisitions(ctx, wf, newly)
	}
	return already, updated, msg, failedLockName, err
}

// isRetryableSyncError reports whether a TryAcquire transaction failure should
// be retried. Matches PostgreSQL SERIALIZABLE conflict (40001), deadlock
// (40P01), and explicit rollback messages by substring against the driver's
// text.
func isRetryableSyncError(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "serialization") ||
		strings.Contains(s, "dependencies") ||
		strings.Contains(s, "deadlock") ||
		strings.Contains(s, "rollback")
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

type acquiredLock struct {
	name string
	kind wfv1.SynchronizationType
}

func (sm *Manager) tryAcquireImpl(ctx context.Context, wf *wfv1.Workflow, tx *sqldb.SessionProxy, holderKey string, failedLockName string, syncItems []*syncItem, lockKeys []string) (bool, bool, string, string, []*acquiredLock, error) {
	defer sm.unlockAll(ctx, lockKeys)
	allAcquirable := true
	msg := ""
	for _, lockKey := range lockKeys {
		lock, found := sm.syncLockMap[lockKey]
		if !found {
			return false, false, "", failedLockName, nil, fmt.Errorf("bug: lock not found: %s", lockKey)
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
		var newly []*acquiredLock
		for i, lockKey := range lockKeys {
			lock := sm.syncLockMap[lockKey]
			var acquired bool
			var acquireErr error
			acquired, msg, acquireErr = lock.tryAcquire(ctx, holderKey, tx)
			if acquireErr != nil {
				// Surface the underlying error so callers (e.g. TryAcquire's
				// retry loop) can decide whether it is retryable. Transient
				// database errors like PostgreSQL SQLSTATE 40001 must reach
				// the retry detector untouched.
				return false, false, "", failedLockName, nil, acquireErr
			}
			if !acquired {
				return false, false, "", failedLockName, nil, fmt.Errorf("bug: failed to acquire something that should have been checked: %s", msg)
			}
			currentHolders, err := sm.getCurrentLockHolders(ctx, lockKey)
			if err != nil {
				return false, false, "", failedLockName, nil, fmt.Errorf("failed to get current lock holders: %w", err)
			}
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockAcquired(holderKey, lockKey, currentHolders) {
				updated = true
				newly = append(newly, &acquiredLock{name: lockKey, kind: syncItems[i].getType()})
			}
		}
		return true, updated, msg, failedLockName, newly, nil
	default: // Not all acquirable
		updated := false
		for i, lockKey := range lockKeys {
			currentHolders, err := sm.getCurrentLockHolders(ctx, lockKey)
			if err != nil {
				return false, false, "", failedLockName, nil, fmt.Errorf("failed to get current lock holders: %w", err)
			}
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockWaiting(holderKey, lockKey, currentHolders) {
				updated = true
			}
		}
		return false, updated, msg, failedLockName, nil, nil
	}
}

func (sm *Manager) recordAcquisitions(ctx context.Context, wf *wfv1.Workflow, newly []*acquiredLock) {
	if sm.metrics == nil || wf == nil || len(newly) == 0 {
		return
	}
	for _, l := range newly {
		switch l.kind {
		case wfv1.SynchronizationTypeMutex:
			sm.metrics.AddMutex(ctx, l.name, wf.Namespace)
		case wfv1.SynchronizationTypeSemaphore:
			sm.metrics.AddSemaphore(ctx, l.name, wf.Namespace)
		}
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

	sm.lock.Lock()
	defer sm.lock.Unlock()

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
				if sm.metrics != nil {
					switch syncItem.getType() {
					case wfv1.SynchronizationTypeMutex:
						sm.metrics.RemoveMutex(ctx, lockKey.String(ctx), wf.Namespace)
					case wfv1.SynchronizationTypeSemaphore:
						sm.metrics.RemoveSemaphore(ctx, lockKey.String(ctx), wf.Namespace)
					}
				}
			}
		}
	}
}

func (sm *Manager) ReleaseAll(ctx context.Context, wf *wfv1.Workflow) bool {
	sm.lock.Lock()
	defer sm.lock.Unlock()

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
				if sm.metrics != nil {
					sm.metrics.RemoveSemaphore(ctx, holding.Semaphore, wf.Namespace)
				}
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
			if sm.metrics != nil {
				sm.metrics.RemoveMutex(ctx, holding.Mutex, wf.Namespace)
			}
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
	switch lock.getKind() {
	case lockKindConfigMap:
		return newInternalSemaphore(ctx, semaphoreName, sm.nextWorkflow, sm.getSyncLimit, sm.syncLimitCacheTTL)
	case lockKindDatabase:
		if sm.dbInfo.SessionProxy == nil {
			return nil, fmt.Errorf("database session is not available for semaphore %s", semaphoreName)
		}
		return newDatabaseSemaphore(ctx, semaphoreName, lock.getDBKey(), sm.nextWorkflow, sm.dbInfo, sm.syncLimitCacheTTL)
	default:
		return nil, fmt.Errorf("invalid lock kind %s when initializing semaphore", lock.getKind())
	}
}

func (sm *Manager) initializeMutex(ctx context.Context, mutexName string) (semaphore, error) {
	lock, err := DecodeLockName(ctx, mutexName)
	if err != nil {
		return nil, err
	}
	switch lock.getKind() {
	case lockKindMutex:
		return newInternalMutex(mutexName, sm.nextWorkflow), nil
	case lockKindDatabase:
		if sm.dbInfo.SessionProxy == nil {
			return nil, fmt.Errorf("database session is not available for mutex %s", mutexName)
		}
		return newDatabaseMutex(mutexName, lock.getDBKey(), sm.nextWorkflow, sm.dbInfo), nil
	default:
		return nil, fmt.Errorf("invalid lock kind %s when initializing mutex", lock.getKind())
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
