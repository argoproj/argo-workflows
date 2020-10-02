package sync

import (
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/v3/errors"
	wfv1 "github.com/argoproj/argo/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/v3/util/slice"
)

type ReleaseNotifyCallbackFunc func(string)
type SyncLimitConfigFunc func(string) (int, error)

type SyncManager struct {
	syncLockMap         map[string]Synchronization
	lock                *sync.Mutex
	releaseNotifyFunc   ReleaseNotifyCallbackFunc
	syncLimitConfigFunc SyncLimitConfigFunc
}

type LockName struct {
	Namespace    string
	Kind         string
	ResourceName string
	Key          string
	Type         LockType
}

type Synchronization interface {
	acquire(holderKey string) bool
	tryAcquire(holderKey string) (bool, string)
	release(key string) bool
	addToQueue(holderKey string, priority int32, creationTime time.Time)
	getCurrentHolders() []string
	getName() string
	getLimit() int
	resize(n int) bool
}

type LockAction string

const (
	LockActionAcquired LockAction = "acquired"
	LockActionReleased LockAction = "released"
	LockActionWaiting  LockAction = "waiting"
)

type LockType string

const (
	LockTypeSemaphore LockType = "semaphore"
	LockTypeMutex     LockType = "mutex"
)

func NewLockManager(getSyncLimitConfigFunc func(string) (int, error), callbackFunc func(string)) *SyncManager {
	return &SyncManager{
		syncLockMap:         make(map[string]Synchronization),
		lock:                &sync.Mutex{},
		releaseNotifyFunc:   callbackFunc,
		syncLimitConfigFunc: getSyncLimitConfigFunc,
	}
}

func (cm *SyncManager) Initialize(wfList *wfv1.WorkflowList) {

	for _, wf := range wfList.Items {
		if wf.Status.Synchronization == nil {
			continue
		}
		if wf.Status.Synchronization.Semaphore != nil {
			for _, holding := range wf.Status.Synchronization.Semaphore.Holding {
				semaphore := cm.syncLockMap[holding.Semaphore]
				if semaphore == nil {
					semaphore, err := cm.initializeSemaphore(holding.Semaphore)
					if err != nil {
						log.Warnf("Synchronization configmap %s is not found. %v", holding.Semaphore, err)
						continue
					}

					cm.syncLockMap[holding.Semaphore] = semaphore
				}
				for _, ele := range holding.Holders {
					resourceKey := getResourceKey(wf.Namespace, wf.Name, ele)
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
	log.Infof("SyncManager initialized successfully")
}

func (cm *SyncManager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := cm.syncLockMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (cm *SyncManager) initializeSemaphore(semaphoreName string) (Synchronization, error) {
	limit, err := cm.syncLimitConfigFunc(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.releaseNotifyFunc, LockTypeSemaphore), nil
}

func (cm *SyncManager) initializeMutex(mutexName string) (Synchronization, error) {
	return NewMutex(mutexName, cm.releaseNotifyFunc), nil
}

func (cm *SyncManager) isSemaphoreSizeChanged(semaphore Synchronization) (bool, int, error) {
	limit, err := cm.syncLimitConfigFunc(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return semaphore.getLimit() != limit, limit, nil
}

func (cm *SyncManager) checkAndUpdateSemaphoreSize(semaphore Synchronization) error {
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
func (cm *SyncManager) TryAcquire(wf *wfv1.Workflow, nodeName string, priority int32, creationTime time.Time, syncLockRef *wfv1.Synchronization) (bool, bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncLockRef == nil {
		return true, false, "", nil
	}

	var lockType LockType
	var err error

	var syncLockName *LockName

	if syncLockRef.Semaphore != nil {
		syncLockName = getSemaphoreLockName(wf.Namespace, syncLockRef.Semaphore)
		semaphoreLockKey := syncLockName.getLockKey()

		semaphoreLock, found := cm.syncLockMap[semaphoreLockKey]
		if !found {
			semaphoreLock, err = cm.initializeSemaphore(semaphoreLockKey)
			if err != nil {
				return false, false, "", err
			}
			cm.syncLockMap[semaphoreLockKey] = semaphoreLock
		}

		// Check syncLock configmap changes
		err := cm.checkAndUpdateSemaphoreSize(semaphoreLock)
		if err != nil {
			return false, false, "", err
		}
		lockType = LockTypeSemaphore
	} else if syncLockRef.Mutex != nil {
		syncLockName = getMutexLockName(wf.Namespace, syncLockRef.Mutex)
		mutexLockKey := syncLockName.getLockKey()
		_, found := cm.syncLockMap[mutexLockKey]
		if !found {
			mutexLock, err := cm.initializeMutex(mutexLockKey)
			if err != nil {
				return false, false, "", err
			}
			cm.syncLockMap[mutexLockKey] = mutexLock
		}
		lockType = LockTypeMutex
	}

	if syncLockName == nil {
		return false, false, "", errors.New(errors.CodeBadRequest, "Requested Synchronization is invalid")
	}

	syncLockKey := syncLockName.getLockKey()
	syncLock, found := cm.syncLockMap[syncLockKey]
	if !found {
		return false, false, "", errors.New(errors.CodeBadRequest, "Requested Synchronized syncLock is invalid")
	}

	holderKey := getHolderKey(wf, nodeName)
	syncLock.addToQueue(holderKey, priority, creationTime)

	status, msg := syncLock.tryAcquire(holderKey)
	if status {
		updated := cm.updateConcurrencyStatus(holderKey, syncLockKey, lockType, LockActionAcquired, wf)
		return true, updated, "", nil
	}

	updated := cm.updateConcurrencyStatus(holderKey, syncLockKey, lockType, LockActionWaiting, wf)
	return false, updated, msg, nil
}

func (cm *SyncManager) Release(wf *wfv1.Workflow, nodeName, namespace string, syncRef *wfv1.Synchronization) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncRef == nil {
		return
	}
	holderKey := getHolderKey(wf, nodeName)

	if syncRef.Semaphore != nil {
		lockName := getSemaphoreLockName(namespace, syncRef.Semaphore)
		if syncLockHolder, ok := cm.syncLockMap[lockName.getLockKey()]; ok {
			syncLockHolder.release(holderKey)
			log.Debugf("%s sync lock is released by %s", lockName.getLockKey(), holderKey)
			cm.updateConcurrencyStatus(holderKey, lockName.getLockKey(), LockTypeSemaphore, LockActionReleased, wf)
		}
	}
	if syncRef.Mutex != nil {
		lockName := getMutexLockName(namespace, syncRef.Mutex)
		if syncLockHolder, ok := cm.syncLockMap[lockName.getLockKey()]; ok {
			syncLockHolder.release(holderKey)
			log.Debugf("%s sync lock is released by %s", lockName.getLockKey(), holderKey)
			cm.updateConcurrencyStatus(holderKey, lockName.getLockKey(), LockTypeMutex, LockActionReleased, wf)
		}
	}
}

func (cm *SyncManager) ReleaseAll(wf *wfv1.Workflow) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if wf.Status.Synchronization == nil {
		return true
	}
	if wf.Status.Synchronization.Semaphore != nil {
		for _, ele := range wf.Status.Synchronization.Semaphore.Holding {
			syncLockHolder := cm.syncLockMap[ele.Semaphore]
			if syncLockHolder == nil {
				continue
			}
			for _, holderName := range ele.Holders {
				resourceKey := getResourceKey(wf.Namespace, wf.Name, holderName)
				syncLockHolder.release(resourceKey)
				cm.updateConcurrencyStatus(holderName, ele.Semaphore, LockTypeSemaphore, LockActionReleased, wf)
				log.Infof("%s released a lock from %s", resourceKey, ele.Semaphore)
			}
		}
		// Clear the Synchronization details
		wf.Status.Synchronization.Semaphore = nil
	}
	if wf.Status.Synchronization.Mutex != nil {
		for _, ele := range wf.Status.Synchronization.Mutex.Holding {
			syncLockHolder := cm.syncLockMap[ele.Mutex]
			if syncLockHolder == nil {
				continue
			}
			resourceKey := getResourceKey(wf.Namespace, wf.Name, ele.Holder)
			syncLockHolder.release(resourceKey)
			cm.updateConcurrencyStatus(ele.Holder, ele.Mutex, LockTypeMutex, LockActionReleased, wf)
			log.Infof("%s released a lock from %s", resourceKey, ele.Mutex)
		}
		wf.Status.Synchronization.Mutex = nil
	}
	wf.Status.Synchronization = nil
	return true
}

// updateSemaphoreStatus updates the semaphore holding and waiting details
func (cm *SyncManager) updateSemaphoreStatus(holderKey, lockKey string, lockAction LockAction, wf *wfv1.Workflow) bool {
	if wf.Status.Synchronization == nil || wf.Status.Synchronization.Semaphore == nil {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Semaphore: &wfv1.SemaphoreStatus{}}
	}
	// Update the semaphore which the workflow is waiting for
	if lockAction == LockActionWaiting {
		index, semaphoreWaiting := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Waiting, lockKey)
		currentHolder := cm.getCurrentLockHolders(lockKey)
		if index == -1 {
			wf.Status.Synchronization.Semaphore.Waiting = append(wf.Status.Synchronization.Semaphore.Waiting, wfv1.SemaphoreHolding{Semaphore: lockKey, Holders: currentHolder})
		} else {
			semaphoreWaiting.Holders = currentHolder
			wf.Status.Synchronization.Semaphore.Waiting[index] = semaphoreWaiting
		}
		return true
	}
	// Update the semaphore which is acquired by the workflow
	if lockAction == LockActionAcquired {
		index, semaphoreHolding := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Holding, lockKey)
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		if index == -1 {
			wf.Status.Synchronization.Semaphore.Holding = append(wf.Status.Synchronization.Semaphore.Holding, wfv1.SemaphoreHolding{Semaphore: lockKey, Holders: []string{holdingName}})
			return true
		} else {
			if !slice.ContainsString(semaphoreHolding.Holders, holdingName) {
				semaphoreHolding.Holders = append(semaphoreHolding.Holders, holdingName)
				wf.Status.Synchronization.Semaphore.Holding[index] = semaphoreHolding
				return true
			}
		}
		return false
	}
	// Clear the semaphore which is released by the workflow
	if lockAction == LockActionReleased {
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		index, semaphoreHolding := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Holding, lockKey)
		if index != -1 {
			semaphoreHolding.Holders = slice.RemoveString(semaphoreHolding.Holders, holdingName)
			wf.Status.Synchronization.Semaphore.Holding[index] = semaphoreHolding
		}
		return true
	}
	return false
}

// updateMutexStatus updates the mutex holding and waiting details
func (cm *SyncManager) updateMutexStatus(holderKey, lockKey string, lockAction LockAction, wf *wfv1.Workflow) bool {
	if wf.Status.Synchronization == nil || wf.Status.Synchronization.Mutex == nil {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Mutex: &wfv1.MutexStatus{}}
	}
	// Update mutex which the workflow is waiting for
	if lockAction == LockActionWaiting {
		index, mutexWaiting := getMutexHolding(wf.Status.Synchronization.Mutex.Waiting, lockKey)
		currentHolder := cm.getCurrentLockHolders(lockKey)
		if len(currentHolder) == 0 {
			return true
		}
		if index == -1 {
			wf.Status.Synchronization.Mutex.Waiting = append(wf.Status.Synchronization.Mutex.Waiting, wfv1.MutexHolding{Mutex: lockKey, Holder: currentHolder[0]})
		} else {
			if mutexWaiting.Holder != currentHolder[0] {
				mutexWaiting.Holder = currentHolder[0]
				wf.Status.Synchronization.Mutex.Waiting[index] = mutexWaiting
			}
		}
		return true
	}
	// Update mutex which is acquired by the workflow
	if lockAction == LockActionAcquired {
		index, mutexHolding := getMutexHolding(wf.Status.Synchronization.Mutex.Holding, lockKey)
		items := strings.Split(holderKey, "/")
		holdingName := items[len(items)-1]
		if index == -1 {
			wf.Status.Synchronization.Mutex.Holding = append(wf.Status.Synchronization.Mutex.Holding, wfv1.MutexHolding{Mutex: lockKey, Holder: holdingName})
			return true
		} else {
			if mutexHolding.Holder != holdingName {
				mutexHolding.Holder = holdingName
				wf.Status.Synchronization.Mutex.Holding[index] = mutexHolding
				return true
			}
		}
		return false
	}
	// Clear the mutex which is released by the workflow
	if lockAction == LockActionReleased {
		index, _ := getMutexHolding(wf.Status.Synchronization.Mutex.Holding, lockKey)
		if index != -1 {
			wf.Status.Synchronization.Mutex.Holding = append(wf.Status.Synchronization.Mutex.Holding[:index], wf.Status.Synchronization.Mutex.Holding[index+1:]...)
		}
		return true
	}
	return false
}

// updateConcurrencyStatus updates the synchronization status update
// It return the status of workflow updated or not.
func (cm *SyncManager) updateConcurrencyStatus(holderKey, lockKey string, lockType LockType, lockAction LockAction, wf *wfv1.Workflow) bool {

	if lockType == LockTypeSemaphore {
		return cm.updateSemaphoreStatus(holderKey, lockKey, lockAction, wf)
	} else if lockType == LockTypeMutex {
		return cm.updateMutexStatus(holderKey, lockKey, lockAction, wf)
	}
	return false
}

func (ln *LockName) getLockKey() string {
	if ln.Kind == string(LockTypeMutex) {
		return fmt.Sprintf("%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName)
	}
	return fmt.Sprintf("%s/%s/%s/%s", ln.Namespace, ln.Kind, ln.ResourceName, ln.Key)
}
func (ln *LockName) validate() error {
	if ln.Namespace == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Namespace is missing")
	}
	if ln.Kind == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Kind is missing")
	}
	if ln.ResourceName == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. ResourceName is missing")
	}
	if ln.Kind != string(LockTypeMutex) && ln.Key == "" {
		return errors.New(errors.CodeBadRequest, "Invalid Lock Key. Key is missing")
	}
	return nil
}

func DecodeLockName(lockName string) (*LockName, error) {
	items := strings.Split(lockName, "/")
	var lock LockName
	// For mutex lockname
	if len(items) == 3 && items[1] == string(LockTypeMutex) {
		lock = LockName{Namespace: items[0], Kind: items[1], ResourceName: items[2]}
	} else if len(items) == 4 { // For Semaphore lockname
		lock = LockName{Namespace: items[0], Kind: items[1], ResourceName: items[2], Key: items[3]}
	} else {
		return nil, errors.New(errors.CodeBadRequest, "Invalid Lock Key")
	}
	err := lock.validate()
	if err != nil {
		return nil, err
	}
	return &lock, nil
}

func NewLockName(namespace, kind, resourceName, lockKey string) *LockName {
	return &LockName{
		Namespace:    namespace,
		Kind:         kind,
		ResourceName: resourceName,
		Key:          lockKey,
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

func getSemaphoreLockName(namespace string, semaphoreRef *wfv1.SemaphoreRef) *LockName {
	if semaphoreRef.ConfigMapKeyRef != nil {
		return NewLockName(namespace, "configmap", semaphoreRef.ConfigMapKeyRef.Name, semaphoreRef.ConfigMapKeyRef.Key)
	}
	return nil
}

func getMutexLockName(namespace string, mutex *wfv1.Mutex) *LockName {
	return NewLockName(namespace, string(LockTypeMutex), mutex.Name, "")
}

func getResourceKey(namespace, wfName, resourceName string) string {
	resourceKey := fmt.Sprintf("%s/%s", namespace, wfName)
	// Template level semaphore
	if resourceName != wfName {
		resourceKey = fmt.Sprintf("%s/%s", resourceKey, resourceName)
	}
	return resourceKey
}

func getSemaphoreHolding(semaphoreHolding []wfv1.SemaphoreHolding, semaphoreName string) (int, wfv1.SemaphoreHolding) {
	for idx, holder := range semaphoreHolding {
		if holder.Semaphore == semaphoreName {
			return idx, holder
		}
	}
	return -1, wfv1.SemaphoreHolding{}
}

func getMutexHolding(mutexHolding []wfv1.MutexHolding, mutexName string) (int, wfv1.MutexHolding) {
	for idx, holder := range mutexHolding {
		if holder.Mutex == mutexName {
			return idx, holder
		}
	}
	return -1, wfv1.MutexHolding{}
}
