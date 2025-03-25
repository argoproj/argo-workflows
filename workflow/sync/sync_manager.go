package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type (
	NextWorkflow      func(string)
	GetSyncLimit      func(string) (int, error)
	IsWorkflowDeleted func(string) bool
)

type Manager struct {
	syncLockMap       map[string]semaphore
	lock              *sync.RWMutex
	nextWorkflow      NextWorkflow
	getSyncLimit      GetSyncLimit
	syncLimitCacheTTL time.Duration
	isWFDeleted       IsWorkflowDeleted
}

func NewLockManager(getSyncLimit GetSyncLimit, syncLimitCacheTTL time.Duration, nextWorkflow NextWorkflow, isWFDeleted IsWorkflowDeleted) *Manager {
	return &Manager{
		syncLockMap:       make(map[string]semaphore),
		lock:              &sync.RWMutex{},
		nextWorkflow:      nextWorkflow,
		getSyncLimit:      getSyncLimit,
		syncLimitCacheTTL: syncLimitCacheTTL,
		isWFDeleted:       isWFDeleted,
	}
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

	log.Debug("Check the workflow existence")
	for _, lock := range sm.syncLockMap {
		keys := lock.getCurrentHolders()
		keys = append(keys, lock.getCurrentPending()...)
		for _, holderKeys := range keys {
			wfKey, err := sm.getWorkflowKey(holderKeys)
			if err != nil {
				continue
			}
			if !sm.isWFDeleted(wfKey) {
				lock.release(holderKeys)
				lock.removeFromQueue(holderKeys)
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
		syncItems, err := allSyncItems(ctx, wf.Spec.Synchronization)
		if err != nil {
			return ErrorLevel, err
		}
		for _, syncItem := range syncItems {
			syncLockName, err := getLockName(syncItem, wf.Namespace)
			if err != nil {
				return ErrorLevel, err
			}
			checkName := syncLockName.encodeName()
			if lockName == checkName {
				return WorkflowLevel, nil
			}
		}
	}

	var lastErr error
	for _, template := range wf.Spec.Templates {
		if template.Synchronization != nil {
			syncItems, err := allSyncItems(ctx, template.Synchronization)
			if err != nil {
				return ErrorLevel, err
			}
			for _, syncItem := range syncItems {
				syncLockName, err := getLockName(syncItem, wf.Namespace)
				if err != nil {
					lastErr = err
					continue
				}
				checkName := syncLockName.encodeName()
				if lockName == checkName {
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
					semaphore, err := sm.initializeSemaphore(holding.Semaphore)
					if err != nil {
						log.Warnf("cannot initialize semaphore '%s': %v", holding.Semaphore, err)
						continue
					}
					sm.syncLockMap[holding.Semaphore] = semaphore
				}

				for _, holders := range holding.Holders {
					level, err := getWorkflowSyncLevelByName(ctx, &wf, holding.Semaphore)
					if err != nil {
						log.Warnf("cannot obtain lock level for '%s' : %v", holding.Semaphore, err)
						continue
					}
					key := getUpgradedKey(&wf, holders, level)
					if semaphore != nil && semaphore.acquire(key) {
						log.Infof("Lock acquired by %s from %s", key, holding.Semaphore)
					}
				}

			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {
				mutex := sm.syncLockMap[holding.Mutex]
				if mutex == nil {
					mutex := sm.initializeMutex(holding.Mutex)
					if holding.Holder != "" {
						level, err := getWorkflowSyncLevelByName(ctx, &wf, holding.Mutex)
						if err != nil {
							log.Warnf("cannot obtain lock level for '%s' : %v", holding.Mutex, err)
							continue
						}
						key := getUpgradedKey(&wf, holding.Holder, level)
						mutex.acquire(key)
					}
					sm.syncLockMap[holding.Mutex] = mutex
				}
			}
		}
	}
	log.Infof("Manager initialized successfully")
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
	syncItems, err := allSyncItems(ctx, syncLockRef)
	if err != nil {
		return false, false, "", failedLockName, fmt.Errorf("requested configuration is invalid: %w", err)
	}
	holderKey := getHolderKey(wf, nodeName)

	var msg string
	lockKeys := make([]string, len(syncItems))
	allAcquirable := true
	for i, syncItem := range syncItems {
		syncLockName, err := getLockName(syncItem, wf.Namespace)
		if err != nil {
			return false, false, "", failedLockName, fmt.Errorf("requested configuration is invalid: %w", err)
		}
		lockKeys[i] = syncLockName.encodeName()
	}
	for i, lockKey := range lockKeys {
		lock, found := sm.syncLockMap[lockKey]
		if !found {
			var err error
			switch syncItems[i].getType() {
			case wfv1.SynchronizationTypeSemaphore:
				lock, err = sm.initializeSemaphore(lockKey)
			case wfv1.SynchronizationTypeMutex:
				lock = sm.initializeMutex(lockKey)
			default:
				return false, false, "", failedLockName, fmt.Errorf("unknown Synchronization Type")
			}
			if err != nil {
				return false, false, "", failedLockName, err
			}
			sm.syncLockMap[lockKey] = lock
		}

		if syncItems[i].getType() == wfv1.SynchronizationTypeSemaphore {
			err := sm.checkAndUpdateSemaphoreSize(lock)
			if err != nil {
				return false, false, "", "", err
			}
		}

		var priority int32
		if wf.Spec.Priority != nil {
			priority = *wf.Spec.Priority
		} else {
			priority = 0
		}
		creationTime := wf.CreationTimestamp
		lock.addToQueue(holderKey, priority, creationTime.Time)

		ensureInit(wf, syncItems[i].getType())
		acquired, already, newMsg := lock.checkAcquire(holderKey)
		if !acquired && !already {
			allAcquirable = false
			if msg == "" {
				msg = newMsg
			}
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
			acquired, msg := lock.tryAcquire(holderKey)
			if !acquired {
				return false, false, "", failedLockName, fmt.Errorf("bug: failed to acquire something that should have been checked: %s", msg)
			}
			currentHolders := sm.getCurrentLockHolders(lockKey)
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockAcquired(holderKey, lockKey, currentHolders) {
				updated = true
			}
		}
		return true, updated, msg, failedLockName, nil
	default: // Not all acquirable
		updated := false
		for i, lockKey := range lockKeys {
			currentHolders := sm.getCurrentLockHolders(lockKey)
			if wf.Status.Synchronization.GetStatus(syncItems[i].getType()).LockWaiting(holderKey, lockKey, currentHolders) {
				updated = true
			}
		}
		return false, updated, msg, failedLockName, nil
	}
}

func (sm *Manager) Release(ctx context.Context, wf *wfv1.Workflow, nodeName string, syncRef *wfv1.Synchronization) {
	if syncRef == nil {
		return
	}

	sm.lock.RLock()
	defer sm.lock.RUnlock()

	holderKey := getHolderKey(wf, nodeName)
	// Ignoring error here is as good as it's going to be, we shouldn't get here as we should
	// should never have acquired anything if this errored
	syncItems, _ := allSyncItems(ctx, syncRef)

	for _, syncItem := range syncItems {
		lockName, err := getLockName(syncItem, wf.Namespace)
		if err != nil {
			return
		}
		if syncLockHolder, ok := sm.syncLockMap[lockName.encodeName()]; ok {
			syncLockHolder.release(holderKey)
			syncLockHolder.removeFromQueue(holderKey)
			lockKey := lockName.encodeName()
			if wf.Status.Synchronization != nil {
				wf.Status.Synchronization.GetStatus(syncItem.getType()).LockReleased(holderKey, lockKey)
			}
		}
	}
}

func (sm *Manager) ReleaseAll(wf *wfv1.Workflow) bool {
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
				syncLockHolder.release(holderKey)
				wf.Status.Synchronization.Semaphore.LockReleased(holderKey, holding.Semaphore)
				log.Infof("%s released a lock from %s", holderKey, holding.Semaphore)
			}
		}

		// Remove the pending Workflow level semaphore keys
		for _, waiting := range wf.Status.Synchronization.Semaphore.Waiting {
			syncLockHolder := sm.syncLockMap[waiting.Semaphore]
			if syncLockHolder == nil {
				continue
			}
			key := getHolderKey(wf, "")
			syncLockHolder.removeFromQueue(key)
		}
		wf.Status.Synchronization.Semaphore = nil
	}

	if wf.Status.Synchronization.Mutex != nil {
		for _, holding := range wf.Status.Synchronization.Mutex.Holding {
			syncLockHolder := sm.syncLockMap[holding.Mutex]
			if syncLockHolder == nil {
				continue
			}

			syncLockHolder.release(holding.Holder)
			wf.Status.Synchronization.Mutex.LockReleased(holding.Holder, holding.Mutex)
			log.Infof("%s released a lock from %s", holding.Holder, holding.Mutex)
		}

		// Remove the pending Workflow level mutex keys
		for _, waiting := range wf.Status.Synchronization.Mutex.Waiting {
			syncLockHolder := sm.syncLockMap[waiting.Mutex]
			if syncLockHolder == nil {
				continue
			}
			key := getHolderKey(wf, "")
			syncLockHolder.removeFromQueue(key)
		}
		wf.Status.Synchronization.Mutex = nil
	}

	for _, node := range wf.Status.Nodes {
		if node.SynchronizationStatus != nil && node.SynchronizationStatus.Waiting != "" {
			lock, ok := sm.syncLockMap[node.SynchronizationStatus.Waiting]
			if ok {
				lock.removeFromQueue(getHolderKey(wf, node.ID))
			}
			node.SynchronizationStatus = nil
			wf.Status.Nodes.Set(node.ID, node)
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

func (sm *Manager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := sm.syncLockMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (sm *Manager) initializeSemaphore(semaphoreName string) (semaphore, error) {
	limit, err := sm.getSyncLimit(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, sm.nextWorkflow, "semaphore"), nil
}

func (sm *Manager) initializeMutex(mutexName string) semaphore {
	return NewMutex(mutexName, sm.nextWorkflow)
}

func (sm *Manager) isSemaphoreSizeChanged(semaphore semaphore) (bool, int, error) {
	limit, err := sm.getSyncLimit(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return semaphore.getLimit() != limit, limit, nil
}

func (sm *Manager) checkAndUpdateSemaphoreSize(semaphore semaphore) error {
	if nowFn().Sub(semaphore.getLimitTimestamp()) < sm.syncLimitCacheTTL {
		return nil
	}

	changed, newLimit, err := sm.isSemaphoreSizeChanged(semaphore)
	if err != nil {
		return err
	}
	if changed {
		semaphore.resize(newLimit)
	} else {
		semaphore.resetLimitTimestamp()
	}
	return nil
}
