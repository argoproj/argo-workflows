package sync

import (
	"context"
	"fmt"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"

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
	syncStorage  syncManagerStorage
}

func NewLockManager(ns string, ki kubernetes.Interface, getSyncLimit GetSyncLimit, nextWorkflow NextWorkflow, isWFDeleted IsWorkflowDeleted) *Manager {
	return &Manager{
		syncLockMap:  make(map[string]Semaphore),
		lock:         &sync.Mutex{},
		nextWorkflow: nextWorkflow,
		getSyncLimit: getSyncLimit,
		isWFDeleted:  isWFDeleted,
		syncStorage:  *newSyncManagerStorage(ns, ki, "argo-sync-storage"),
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
	for key, lock := range cm.syncLockMap {
		keys := lock.getCurrentHolders()
		keys = append(keys, lock.getCurrentPending()...)
		for _, holderKeys := range keys {
			wfKey, err := cm.getWorkflowKey(holderKeys)
			if err != nil {
				continue
			}
			if !cm.isWFDeleted(wfKey) {
				release(cm, key, []string{holderKeys})
			}
		}
	}
}

func (cm *Manager) Initialize(wfs []wfv1.Workflow) {
	for _, wf := range wfs {
		if wf.Status.Synchronization == nil {
			continue
		}

		if wf.Status.Synchronization.Semaphore != nil {
			for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
				semaphore := cm.syncLockMap[holding.Semaphore]
				entry, _, err := cm.syncStorage.Load(context.Background(), holding.Semaphore)
				if err != nil || entry == nil {
					continue
				}

				log.Debugln("HOLDING VALUE [Semaphore] IS ", holding.Semaphore)
				if semaphore == nil {
					semaphore, err = cm.initializeSemaphore(holding.Semaphore)
					if err != nil {
						log.Warnf("cannot initialize semaphore '%s': %v", holding.Semaphore, err)
						continue
					}
					cm.syncLockMap[holding.Semaphore] = semaphore
				}

				for _, holder := range entry.Holders {
					semaphore.acquire(holder)
					log.Infof("Lock acquired by %s from %s", holder, holding.Semaphore)
				}
			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {
				log.Debugln("HOLDING VALUE [Mutex] IS ", holding.Mutex)
				mutex := cm.syncLockMap[holding.Mutex]
				entry, _, err := cm.syncStorage.Load(context.Background(), holding.Mutex)
				if err != nil {
					log.Errorf("Skipping acquiring mutex for %s duie to %s", holding.Mutex, err.Error())
					continue
				}
				if entry == nil {
					log.Errorf("Could not find entry for %s", holding.Mutex)
					continue
				}

				if len(entry.Holders) != 1 {
					log.Warnf("Expected 1 holder but got %d, skipping", len(entry.Holders))
					continue
				}

				if mutex == nil {
					mutex = cm.initializeMutex(holding.Mutex)
					cm.syncLockMap[holding.Mutex] = mutex
				}
				mutex.acquire(entry.Holders[0])
				log.Infof("Lock acquired by %s from %s", entry.Holders[0], holding.Mutex)
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
	log.Debugln("TryAcquire LockName is ", lockKey)
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
		holders := []string{}
		switch syncLockRef.GetType() {
		case v1alpha1.SynchronizationTypeMutex:
			holders = []string{holderKey}
		case v1alpha1.SynchronizationTypeSemaphore:
			var meta *SyncMetadataEntry
			var found bool
			meta, found, err = cm.syncStorage.Load(context.Background(), lockKey)
			// entry was not present
			if found && err == nil {
				holders = append(meta.Holders, holderKey)
			} else {
				holders = []string{holderKey}
			}
			holders = []string{}
		default:
			err = fmt.Errorf("Unknown SynchronizationType of %s", syncLockRef.GetType())
		}
		// handle potential error from switch statement
		if err != nil {
			lock.release(holderKey)
			lock.removeFromQueue(holderKey)
			return false, false, "", err
		}

		err = cm.syncStorage.Store(context.Background(), lockKey, holders, syncLockRef.GetType())
		if err != nil {
			release(cm, lockKey, []string{holderKey})
			return false, false, "", err
		}
		updated := wf.Status.Synchronization.GetStatus(syncLockRef.GetType()).LockAcquired(holderKey, lockKey, currentHolders)
		return acquired, updated, "", nil
	}

	updated := wf.Status.Synchronization.GetStatus(syncLockRef.GetType()).LockWaiting(holderKey, lockKey, currentHolders)
	return false, updated, msg, nil
}

// Always called inside another function with a mutex acquired
func release(cm *Manager, key string, holders []string) error {
	lock := cm.syncLockMap[key]
	lockCurrentHolders := make(map[string]bool)
	for _, holder := range lock.getCurrentHolders() {
		lockCurrentHolders[holder] = true
	}

	// Ensure all are valid holders, otherwise we fail
	for _, holder := range holders {
		_, ok := lockCurrentHolders[holder]
		if !ok {
			return fmt.Errorf("%s is not a valid holder of %s", holder, key)
		}
	}

	for _, holder := range holders {
		lock.release(holder)
		lock.removeFromQueue(holder)
	}

	return cm.syncStorage.DeleteLockHolders(context.Background(), key, holders)
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
		cm.syncStorage.DeleteLockHolders(context.Background(), lockName.EncodeName(), []string{holderKey})
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

			err := cm.syncStorage.DeleteLock(context.Background(), holding.Semaphore)
			if err != nil {
				continue
			}
			delete(cm.syncLockMap, holding.Semaphore)
		}
		// Remove the pending Workflow level semaphore keys
		for _, waiting := range wf.Status.Synchronization.Semaphore.Waiting {
			syncLockHolder := cm.syncLockMap[waiting.Semaphore]
			if syncLockHolder == nil {
				continue
			}
			err := cm.syncStorage.DeleteLock(context.Background(), waiting.Semaphore)
			if err == nil {
				continue
			}
			delete(cm.syncLockMap, waiting.Semaphore)
		}
		wf.Status.Synchronization.Semaphore = nil
	}

	// TODO REST OF RELASING
	if wf.Status.Synchronization.Mutex != nil {
		for _, holding := range wf.Status.Synchronization.Mutex.Holding {
			syncLockHolder := cm.syncLockMap[holding.Mutex]
			if syncLockHolder == nil {
				continue
			}

			resourceKey := getResourceKey(wf.Namespace, wf.Name, holding.Holder)
			syncLockHolder.release(resourceKey)
			wf.Status.Synchronization.Mutex.LockReleased(holding.Holder, holding.Mutex)
			log.Infof("%s released a lock from %s", resourceKey, holding.Mutex)
		}

		// Remove the pending Workflow level mutex keys
		for _, waiting := range wf.Status.Synchronization.Mutex.Waiting {
			syncLockHolder := cm.syncLockMap[waiting.Mutex]
			if syncLockHolder == nil {
				continue
			}
			resourceKey := getResourceKey(wf.Namespace, wf.Name, wf.Name)
			syncLockHolder.removeFromQueue(resourceKey)
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
			wf.Status.Nodes[node.ID] = node
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
