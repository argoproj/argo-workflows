package sync

import (
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type LockReleased func(string)
type GetSyncLimit func(string) (int, error)

type Manager struct {
	syncLockMap  map[string]Synchronization
	lock         *sync.Mutex
	lockReleased LockReleased
	getSyncLimit GetSyncLimit
}

//type LockAction string

//const (
//	LockActionAcquired LockAction = "acquired"
//	LockActionReleased LockAction = "released"
//	LockActionWaiting  LockAction = "waiting"
//)

// TODO: try to get rid of this
type LockType string

const (
	LockTypeSemaphore LockType = "semaphore"
	LockTypeMutex     LockType = "mutex"
)

func NewLockManager(getSyncLimit GetSyncLimit, lockReleased LockReleased) *Manager {
	return &Manager{
		syncLockMap:  make(map[string]Synchronization),
		lock:         &sync.Mutex{},
		lockReleased: lockReleased,
		getSyncLimit: getSyncLimit,
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
				if semaphore == nil {
					semaphore, err := cm.initializeSemaphore(holding.Semaphore)
					if err != nil {
						log.Warnf("cannot initialize semaphore '%s': %v", holding.Semaphore, err)
						continue
					}
					cm.syncLockMap[holding.Semaphore] = semaphore
				}

				for _, holders := range holding.Holders {
					resourceKey := getResourceKey(wf.Namespace, wf.Name, holders)
					if semaphore != nil && semaphore.acquire(resourceKey) {
						log.Infof("Lock acquired by %s from %s", resourceKey, holding.Semaphore)
					}
				}
			}
		}

		if wf.Status.Synchronization.Mutex != nil {
			for _, holding := range wf.Status.Synchronization.Mutex.Holding {

				mutex := cm.syncLockMap[holding.Mutex]
				if mutex == nil {
					mutex, err := cm.initializeMutex(holding.Mutex)
					if err != nil {
						log.Warnf("Synchronization Mutex %s initialization failed. %v", holding.Mutex, err)
						continue
					}
					if holding.Holder != "" {
						resourceKey := getResourceKey(wf.Namespace, wf.Name, holding.Holder)
						mutex.acquire(resourceKey)
					}
					cm.syncLockMap[holding.Mutex] = mutex
				}
			}
		}
	}
	log.Infof("Manager initialized successfully")
}

func (cm *Manager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := cm.syncLockMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (cm *Manager) initializeSemaphore(semaphoreName string) (Synchronization, error) {
	limit, err := cm.getSyncLimit(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.lockReleased, LockTypeSemaphore), nil
}

func (cm *Manager) initializeMutex(mutexName string) (Synchronization, error) {
	return NewMutex(mutexName, cm.lockReleased), nil
}

func (cm *Manager) isSemaphoreSizeChanged(semaphore Synchronization) (bool, int, error) {
	limit, err := cm.getSyncLimit(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return semaphore.getLimit() != limit, limit, nil
}

func (cm *Manager) checkAndUpdateSemaphoreSize(semaphore Synchronization) error {
	changed, newLimit, err := cm.isSemaphoreSizeChanged(semaphore)
	if err != nil {
		return err
	}
	if changed {
		semaphore.resize(newLimit)
	}
	return nil
}

// TryAcquire tries to acquire the lock from semaphore.
// It returns status of acquiring a lock , status of Workflow status updated, waiting message if lock is not available and any error encountered
func (cm *Manager) TryAcquire(wf *wfv1.Workflow, nodeName string, priority int32, creationTime time.Time, syncLockRef *wfv1.Synchronization) (bool, bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncLockRef == nil {
		return true, false, "", nil
	}

	syncLockName, err := GetLockName(syncLockRef, wf.Namespace)
	if err != nil {
		return false, false, "", fmt.Errorf("requested configuration is invalid: %w", err)
	}

	lockKey := syncLockName.getLockKey()
	lock, found := cm.syncLockMap[lockKey]
	if !found {
		switch syncLockRef.GetType() {
		case wfv1.SynchronizationTypeSemaphore:
			lock, err = cm.initializeSemaphore(lockKey)
		case wfv1.SynchronizationTypeMutex:
			lock, err = cm.initializeMutex(lockKey)
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
	lock.addToQueue(holderKey, priority, creationTime)

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
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncRef == nil {
		return
	}

	holderKey := getHolderKey(wf, nodeName)
	lockName, err := GetLockName(syncRef, wf.Namespace)
	if err != nil {
		return
	}

	if syncLockHolder, ok := cm.syncLockMap[lockName.getLockKey()]; ok {
		syncLockHolder.release(holderKey)
		log.Debugf("%s sync lock is released by %s", lockName.getLockKey(), holderKey)
		lockKey := lockName.getLockKey()
		wf.Status.Synchronization.GetStatus(syncRef.GetType()).LockReleased(holderKey, lockKey)
	}

	//if syncRef.Semaphore != nil {
	//	lockName := getSemaphoreLockName(wf.Namespace, syncRef.Semaphore)
	//	if syncLockHolder, ok := cm.syncLockMap[lockName.getLockKey()]; ok {
	//		syncLockHolder.release(holderKey)
	//		log.Debugf("%s sync lock is released by %s", lockName.getLockKey(), holderKey)
	//		cm.updateConcurrencyStatus(holderKey, lockName.getLockKey(), LockTypeSemaphore, LockActionReleased, wf)
	//	}
	//}
	//if syncRef.Mutex != nil {
	//	lockName := getMutexLockName(wf.Namespace, syncRef.Mutex)
	//	if syncLockHolder, ok := cm.syncLockMap[lockName.getLockKey()]; ok {
	//		syncLockHolder.release(holderKey)
	//		log.Debugf("%s sync lock is released by %s", lockName.getLockKey(), holderKey)
	//		cm.updateConcurrencyStatus(holderKey, lockName.getLockKey(), LockTypeMutex, LockActionReleased, wf)
	//	}
	//}
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
				resourceKey := getResourceKey(wf.Namespace, wf.Name, holderKey)
				syncLockHolder.release(resourceKey)
				wf.Status.Synchronization.Semaphore.LockReleased(holderKey, holding.Semaphore)
				log.Infof("%s released a lock from %s", resourceKey, holding.Semaphore)
			}
		}
		wf.Status.Synchronization.Semaphore = nil
	}

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
		wf.Status.Synchronization.Mutex = nil
	}

	wf.Status.Synchronization = nil
	return true
}

func ensureInit(wf *wfv1.Workflow, lockType wfv1.SynchronizationType) {
	if lockType == wfv1.SynchronizationTypeSemaphore && (wf.Status.Synchronization == nil || wf.Status.Synchronization.Semaphore == nil) {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Semaphore: &wfv1.SemaphoreStatus{}}
	}
	if lockType == wfv1.SynchronizationTypeMutex && (wf.Status.Synchronization == nil || wf.Status.Synchronization.Mutex == nil) {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Mutex: &wfv1.MutexStatus{}}
	}
}

// updateSemaphoreStatus updates the semaphore holding and waiting details
//func (cm *Manager) updateSemaphoreStatus(holderKey, lockKey string, lockAction LockAction, wf *wfv1.Workflow) bool {
//	if wf.Status.Synchronization == nil || wf.Status.Synchronization.Semaphore == nil {
//		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Semaphore: &wfv1.SemaphoreStatus{}}
//	}
//	Update the semaphore which the workflow is waiting for
//	if lockAction == LockActionWaiting {
//		index, semaphoreWaiting := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Waiting, lockKey)
//		currentHolder := cm.getCurrentLockHolders(lockKey)
//		if index == -1 {
//			wf.Status.Synchronization.Semaphore.Waiting = append(wf.Status.Synchronization.Semaphore.Waiting, wfv1.SemaphoreHolding{Semaphore: lockKey, Holders: currentHolder})
//		} else {
//			semaphoreWaiting.Holders = currentHolder
//			wf.Status.Synchronization.Semaphore.Waiting[index] = semaphoreWaiting
//		}
//		return true
//	}
//	Update the semaphore which is acquired by the workflow
//	if lockAction == LockActionAcquired {
//		index, semaphoreHolding := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Holding, lockKey)
//		items := strings.Split(holderKey, "/")
//		holdingName := items[len(items)-1]
//		if index == -1 {
//			wf.Status.Synchronization.Semaphore.Holding = append(wf.Status.Synchronization.Semaphore.Holding, wfv1.SemaphoreHolding{Semaphore: lockKey, Holders: []string{holdingName}})
//			return true
//		} else {
//			if !slice.ContainsString(semaphoreHolding.Holders, holdingName) {
//				semaphoreHolding.Holders = append(semaphoreHolding.Holders, holdingName)
//				wf.Status.Synchronization.Semaphore.Holding[index] = semaphoreHolding
//				return true
//			}
//		}
//		return false
//	}
//	Clear the semaphore which is released by the workflow
//	if lockAction == LockActionReleased {
//		items := strings.Split(holderKey, "/")
//		holdingName := items[len(items)-1]
//		index, semaphoreHolding := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Holding, lockKey)
//		if index != -1 {
//			semaphoreHolding.Holders = slice.RemoveString(semaphoreHolding.Holders, holdingName)
//			wf.Status.Synchronization.Semaphore.Holding[index] = semaphoreHolding
//		}
//		return true
//	}
//	return false
//}

// updateMutexStatus updates the mutex holding and waiting details
//func (cm *Manager) updateMutexStatus(holderKey, lockKey string, lockAction LockAction, wf *wfv1.Workflow) bool {
//	if wf.Status.Synchronization == nil || wf.Status.Synchronization.Mutex == nil {
//		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Mutex: &wfv1.MutexStatus{}}
//	}
//	Update mutex which the workflow is waiting for
//	if lockAction == LockActionWaiting {
//		index, mutexWaiting := getMutexHolding(wf.Status.Synchronization.Mutex.Waiting, lockKey)
//		currentHolder := cm.getCurrentLockHolders(lockKey)
//		if len(currentHolder) == 0 {
//			return true
//		}
//		if index == -1 {
//			wf.Status.Synchronization.Mutex.Waiting = append(wf.Status.Synchronization.Mutex.Waiting, wfv1.MutexHolding{Mutex: lockKey, Holder: currentHolder[0]})
//		} else {
//			if mutexWaiting.Holder != currentHolder[0] {
//				mutexWaiting.Holder = currentHolder[0]
//				wf.Status.Synchronization.Mutex.Waiting[index] = mutexWaiting
//			}
//		}
//		return true
//	}
//
//	Update mutex which is acquired by the workflow
//	if lockAction == LockActionAcquired {
//		index, mutexHolding := getMutexHolding(wf.Status.Synchronization.Mutex.Holding, lockKey)
//		items := strings.Split(holderKey, "/")
//		holdingName := items[len(items)-1]
//		if index == -1 {
//			wf.Status.Synchronization.Mutex.Holding = append(wf.Status.Synchronization.Mutex.Holding, wfv1.MutexHolding{Mutex: lockKey, Holder: holdingName})
//			return true
//		} else {
//			if mutexHolding.Holder != holdingName {
//				mutexHolding.Holder = holdingName
//				wf.Status.Synchronization.Mutex.Holding[index] = mutexHolding
//				return true
//			}
//		}
//		return false
//	}
//	Clear the mutex which is released by the workflow
//	if lockAction == LockActionReleased {
//		index, _ := getMutexHolding(wf.Status.Synchronization.Mutex.Holding, lockKey)
//		if index != -1 {
//			wf.Status.Synchronization.Mutex.Holding = append(wf.Status.Synchronization.Mutex.Holding[:index], wf.Status.Synchronization.Mutex.Holding[index+1:]...)
//		}
//		return true
//	}
//	return false
//}

// updateConcurrencyStatus updates the synchronization status update
// It return the status of workflow updated or not.
//func (cm *Manager) updateConcurrencyStatus(holderKey, lockKey string, lockType LockType, lockAction LockAction, wf *wfv1.Workflow) bool {
//	if lockType == LockTypeSemaphore {
//		return cm.updateSemaphoreStatus(holderKey, lockKey, lockAction, wf)
//	} else if lockType == LockTypeMutex {
//		return cm.updateMutexStatus(holderKey, lockKey, lockAction, wf)
//	}
//	return false
//}

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

//func getSemaphoreLockName(namespace string, semaphoreRef *wfv1.SemaphoreRef) *LockName {
//	if semaphoreRef.ConfigMapKeyRef != nil {
//		return NewLockName(namespace, semaphoreRef.ConfigMapKeyRef.Name, semaphoreRef.ConfigMapKeyRef.Key, LockKindConfigMap)
//	}
//	return nil
//}
//
//func getMutexLockName(namespace string, mutex *wfv1.Mutex) *LockName {
//	return NewLockName(namespace, mutex.Name, "", LockKindMutex)
//}

func getResourceKey(namespace, wfName, resourceName string) string {
	resourceKey := fmt.Sprintf("%s/%s", namespace, wfName)
	// Template level semaphore
	if resourceName != wfName {
		resourceKey = fmt.Sprintf("%s/%s", resourceKey, resourceName)
	}
	return resourceKey
}

// Safe to delete these to funcs
//func getSemaphoreHolding(semaphoreHolding []wfv1.SemaphoreHolding, semaphoreName string) (int, wfv1.SemaphoreHolding) {
//	for idx, holder := range semaphoreHolding {
//		if holder.Semaphore == semaphoreName {
//			return idx, holder
//		}
//	}
//	return -1, wfv1.SemaphoreHolding{}
//}
//
//func getMutexHolding(mutexHolding []wfv1.MutexHolding, mutexName string) (int, wfv1.MutexHolding) {
//	for idx, holder := range mutexHolding {
//		if holder.Mutex == mutexName {
//			return idx, holder
//		}
//	}
//	return -1, wfv1.MutexHolding{}
//}
