package sync

import (
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

type (
	NextWorkflow      func(string)
	GetSyncLimit      func(string) (int, error)
	IsWorkflowDeleted func(string) bool
)

type Manager struct {
	syncLockMap  map[string]Semaphore
	lock         *sync.Mutex
	nextWorkflow NextWorkflow
	getSyncLimit GetSyncLimit
	isWFDeleted  IsWorkflowDeleted
}

func NewLockManager(getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, isWFDeleted IsWorkflowDeleted) *Manager {
	return &Manager{
		syncLockMap:  make(map[string]Semaphore),
		lock:         &sync.Mutex{},
		nextWorkflow: nextWorkflow,
		getSyncLimit: getSyncLimit,
		isWFDeleted:  isWFDeleted,
	}
}

func (cm *Manager) getWorkflowKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("holderkey is empty")
	}
	items := strings.Split(key, "/")
	if len(items) < 2 {
		return "", fmt.Errorf("invalid holderkey format")
	}
	return fmt.Sprintf("%s/%s", items[0], items[1]), nil
}

func (cm *Manager) CheckWorkflowExistence() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	log.Debug("Check the workflow existence")
	for _, lock := range cm.syncLockMap {
		keys := lock.getCurrentHolders()
		keys = append(keys, lock.getCurrentPending()...)
		for _, holderKeys := range keys {
			wfKey, err := cm.getWorkflowKey(holderKeys)
			if err != nil {
				continue
			}
			if !cm.isWFDeleted(wfKey) {
				lock.release(holderKeys)
				lock.removeFromQueue(holderKeys)
			}
		}
	}
}

func getUpgradedKey(wf *wfv1.Workflow, key string) string {
	if wfv1.CheckHolderKeyVersion(key) == wfv1.HoldingNameV1 {
		return getHolderKey(wf, key)
	}
	return key
}

func (cm *Manager) Initialize(wfs []wfv1.Workflow) {
	for _, wf := range wfs {
		if wf.Status.Synchronization == nil {
			continue
		}

		if wf.Status.Synchronization.Semaphore != nil {
			for _, holding := range wf.Status.Synchronization.Semaphore.Holding {

				semaphore := cm.syncLockMap[holding.Semaphore]
				if semaphore == nil {
					semaphore, err := cm.initializeSemaphore(holding.Semaphore)
					if err != nil {
						log.Warnf("cannot initialize semaphore '%s': %v", holding.Semaphore, err)
						continue
					}
					cm.syncLockMap[holding.Semaphore] = semaphore
				}

				for _, holders := range holding.Holders {
					key := getUpgradedKey(&wf, holders)
					if semaphore != nil && semaphore.acquire(key) {
						log.Infof("Lock acquired by %s from %s", key, holding.Semaphore)
					}
				}

			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {

				mutex := cm.syncLockMap[holding.Mutex]
				if mutex == nil {
					mutex := cm.initializeMutex(holding.Mutex)
					if holding.Holder != "" {
						key := getUpgradedKey(&wf, holding.Holder)
						mutex.acquire(key)
					}
					cm.syncLockMap[holding.Mutex] = mutex
				}
			}
		}
	}
	log.Infof("Manager initialized successfully")
}

// TryAcquire tries to acquire the lock from semaphore.
// It returns status of acquiring a lock , status of Workflow status updated, waiting message if lock is not available and any error encountered
func (cm *Manager) TryAcquire(wf *wfv1.Workflow, nodeName string, syncLockRef *wfv1.Synchronization) (bool, bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncLockRef == nil {
		return false, false, "", fmt.Errorf("cannot acquire lock from nil Synchronization")
	}

	syncLockName, err := GetLockName(syncLockRef, wf.Namespace)
	if err != nil {
		return false, false, "", fmt.Errorf("requested configuration is invalid: %w", err)
	}

	lockKey := syncLockName.EncodeName()
	lock, found := cm.syncLockMap[lockKey]
	if !found {
		switch syncLockRef.GetType() {
		case wfv1.SynchronizationTypeSemaphore:
			lock, err = cm.initializeSemaphore(lockKey)
		case wfv1.SynchronizationTypeMutex:
			lock = cm.initializeMutex(lockKey)
		default:
			return false, false, "", fmt.Errorf("unknown Synchronization Type")
		}
		if err != nil {
			return false, false, "", err
		}
		cm.syncLockMap[lockKey] = lock
	}

	if syncLockRef.GetType() == wfv1.SynchronizationTypeSemaphore {
		err := cm.checkAndUpdateSemaphoreSize(lock)
		if err != nil {
			return false, false, "", err
		}
	}

	holderKey := getHolderKey(wf, nodeName)
	var priority int32
	if wf.Spec.Priority != nil {
		priority = *wf.Spec.Priority
	} else {
		priority = 0
	}
	creationTime := wf.CreationTimestamp
	lock.addToQueue(holderKey, priority, creationTime.Time)

	ensureInit(wf, syncLockRef.GetType())
	currentHolders := cm.getCurrentLockHolders(lockKey)
	acquired, msg := lock.tryAcquire(holderKey)
	if acquired {
		updated := wf.Status.Synchronization.GetStatus(syncLockRef.GetType()).LockAcquired(holderKey, lockKey, currentHolders)
		return true, updated, "", nil
	}

	updated := wf.Status.Synchronization.GetStatus(syncLockRef.GetType()).LockWaiting(holderKey, lockKey, currentHolders)
	return false, updated, msg, nil
}

func (cm *Manager) Release(wf *wfv1.Workflow, nodeName string, syncRef *wfv1.Synchronization) {
	if syncRef == nil {
		return
	}

	cm.lock.Lock()
	defer cm.lock.Unlock()

	holderKey := getHolderKey(wf, nodeName)
	lockName, err := GetLockName(syncRef, wf.Namespace)
	if err != nil {
		return
	}

	if syncLockHolder, ok := cm.syncLockMap[lockName.EncodeName()]; ok {
		syncLockHolder.release(holderKey)
		syncLockHolder.removeFromQueue(holderKey)
		log.Debugf("%s sync lock is released by %s", lockName.EncodeName(), holderKey)
		lockKey := lockName.EncodeName()
		if wf.Status.Synchronization != nil {
			wf.Status.Synchronization.GetStatus(syncRef.GetType()).LockReleased(holderKey, lockKey)
		}
	}
}

func (cm *Manager) ReleaseAll(wf *wfv1.Workflow) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if wf.Status.Synchronization == nil {
		return true
	}

	if wf.Status.Synchronization.Semaphore != nil {
		for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
			syncLockHolder := cm.syncLockMap[holding.Semaphore]
			if syncLockHolder == nil {
				continue
			}

			for _, holderKey := range holding.Holders {
				logKey := ""
				if v1alpha1.CheckHolderKeyVersion(holderKey) == v1alpha1.HoldingNameV1 {
					resourceKey := getResourceKey(wf.Namespace, wf.Name, holderKey)
					logKey = resourceKey
					syncLockHolder.release(resourceKey)
				} else {
					logKey = holderKey
					syncLockHolder.release(holderKey)
				}
				wf.Status.Synchronization.Semaphore.LockReleased(holderKey, holding.Semaphore)
				log.Infof("%s released a lock from %s", logKey, holding.Semaphore)
			}
		}

		// Remove the pending Workflow level semaphore keys
		for _, waiting := range wf.Status.Synchronization.Semaphore.Waiting {
			syncLockHolder := cm.syncLockMap[waiting.Semaphore]
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
			syncLockHolder := cm.syncLockMap[holding.Mutex]
			if syncLockHolder == nil {
				continue
			}

			key := ""
			if v1alpha1.CheckHolderKeyVersion(holding.Holder) == v1alpha1.HoldingNameV1 {
				resourceKey := getResourceKey(wf.Namespace, wf.Name, holding.Holder)
				key = resourceKey
			} else {
				key = holding.Holder
			}

			syncLockHolder.release(key)
			wf.Status.Synchronization.Mutex.LockReleased(holding.Holder, holding.Mutex)
			log.Infof("%s released a lock from %s", key, holding.Mutex)
		}

		// Remove the pending Workflow level mutex keys
		for _, waiting := range wf.Status.Synchronization.Mutex.Waiting {
			syncLockHolder := cm.syncLockMap[waiting.Mutex]
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
			lock, ok := cm.syncLockMap[node.SynchronizationStatus.Waiting]
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

// DEPRECATED: To be removed in 3.7
// This function (incorrectly) tries to reconstruct the holderKey.
// the new holderKey provides enough information that reconstruction is not needed.
func getResourceKey(namespace, wfName, resourceName string) string {
	resourceKey := fmt.Sprintf("%s/%s", namespace, wfName)
	if resourceName != wfName {
		resourceKey = fmt.Sprintf("%s/%s", resourceKey, resourceName)
	}
	return resourceKey
}

func (cm *Manager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := cm.syncLockMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (cm *Manager) initializeSemaphore(semaphoreName string) (Semaphore, error) {
	limit, err := cm.getSyncLimit(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.nextWorkflow, "semaphore"), nil
}

func (cm *Manager) initializeMutex(mutexName string) Semaphore {
	return NewMutex(mutexName, cm.nextWorkflow)
}

func (cm *Manager) isSemaphoreSizeChanged(semaphore Semaphore) (bool, int, error) {
	limit, err := cm.getSyncLimit(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return semaphore.getLimit() != limit, limit, nil
}

func (cm *Manager) checkAndUpdateSemaphoreSize(semaphore Semaphore) error {
	changed, newLimit, err := cm.isSemaphoreSizeChanged(semaphore)
	if err != nil {
		return err
	}
	if changed {
		semaphore.resize(newLimit)
	}
	return nil
}
