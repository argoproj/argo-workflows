package sync

import (
	"fmt"

	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/util/slice"
)

type ReleaseNotifyCallbackFunc func(string)
type SyncLimitConfigFunc func(string) (int, error)

type SyncManager struct {
	//kubeClient          kubernetes.Interface
	syncLockMap         map[string]Synchronization
	lock                *sync.Mutex
	releaseNotifyFunc   ReleaseNotifyCallbackFunc
	syncLimitConfigFunc SyncLimitConfigFunc
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
	acquired LockAction = "acquired"
	released LockAction = "released"
	waiting  LockAction = "waiting"
)

type LockType string

const (
	TypeSemaphore LockType = "semaphore"
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
		if wf.Status.Synchronization == nil || wf.Status.Synchronization.Semaphore == nil || wf.Status.Synchronization.Semaphore.Holding == nil {
			continue
		}
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
	return NewSemaphore(semaphoreName, limit, cm.releaseNotifyFunc), nil
}

func (cm *SyncManager) isSemaphoreSizeChanged(semaphore Synchronization) (bool, int, error) {
	limit, err := cm.syncLimitConfigFunc(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return !(semaphore.getLimit() == limit), limit, nil
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
// It returns status of acquiring a lock , status of Workflow status updated,  waiting message if lock is not available and any error encountered
func (cm *SyncManager) TryAcquire(wf *wfv1.Workflow, nodeName string, priority int32, creationTime time.Time, syncLockRef *wfv1.Synchronization) (bool, bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if syncLockRef == nil {
		return true, false, "", nil
	}

	var lockType LockType
	var err error

	lockKey := ""

	if syncLockRef.Semaphore != nil {
		lockKey = getSemaphoreKey(wf.Namespace, syncLockRef.Semaphore)
		semaphoreLock, found := cm.syncLockMap[lockKey]

		if !found {
			semaphoreLock, err = cm.initializeSemaphore(lockKey)
			if err != nil {
				return false, false, "", err
			}
			cm.syncLockMap[lockKey] = semaphoreLock
		}

		// Check lock configmap changes
		err := cm.checkAndUpdateSemaphoreSize(semaphoreLock)

		if err != nil {
			return false, false, "", err
		}
		lockType = TypeSemaphore
	}
	if lockKey == "" {
		return false, false, "", errors.New(errors.CodeBadRequest, "Requested Synchronization is invalid")
	}

	lock, found := cm.syncLockMap[lockKey]
	if !found {
		return false, false, "", errors.New(errors.CodeBadRequest, "Requested Synchronized lock is invalid")
	}

	holderKey := getHolderKey(wf, nodeName)

	lock.addToQueue(holderKey, priority, creationTime)

	status, msg := lock.tryAcquire(holderKey)
	if status {
		updated := cm.updateConcurrencyStatus(holderKey, lockKey, lockType, acquired, wf)
		return true, updated, "", nil
	}

	updated := cm.updateConcurrencyStatus(holderKey, lockKey, lockType, waiting, wf)
	return false, updated, msg, nil
}

func (cm *SyncManager) Release(wf *wfv1.Workflow, nodeName, namespace string, concurrencyRef *wfv1.Synchronization) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if concurrencyRef == nil {
		return
	}
	holderKey := getHolderKey(wf, nodeName)
	if concurrencyRef.Semaphore != nil {
		concurrencyKey := getSemaphoreKey(namespace, concurrencyRef.Semaphore)
		if concurrency, ok := cm.syncLockMap[concurrencyKey]; ok {
			concurrency.release(holderKey)
			log.Debugf("%s sync lock is released by %s", concurrencyKey, holderKey)
			cm.updateConcurrencyStatus(holderKey, concurrencyKey, TypeSemaphore, released, wf)
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
			concurrency := cm.syncLockMap[ele.Semaphore]
			if concurrency == nil {
				continue
			}
			for _, holderName := range ele.Holders {
				resourceKey := getResourceKey(wf.Namespace, wf.Name, holderName)
				concurrency.release(resourceKey)
				cm.updateConcurrencyStatus(holderName, ele.Semaphore, TypeSemaphore, released, wf)
				log.Infof("%s released a lock from %s", resourceKey, ele.Semaphore)
			}
		}
		// Clear the Synchronization details
		wf.Status.Synchronization.Semaphore = nil
		wf.Status.Synchronization = nil
	}
	return true
}

// updateConcurrencyStatus updates the synchronization status update
// It return the status of workflow updated or not.
func (cm *SyncManager) updateConcurrencyStatus(holderKey, lockKey string, lockType LockType, lockAction LockAction, wf *wfv1.Workflow) bool {

	if wf.Status.Synchronization == nil {
		wf.Status.Synchronization = &wfv1.SynchronizationStatus{Semaphore: &wfv1.SemaphoreStatus{}}
	}
	if lockType == TypeSemaphore {
		if lockAction == waiting {
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

		if lockAction == acquired {
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
		if lockAction == released {
			items := strings.Split(holderKey, "/")
			holdingName := items[len(items)-1]
			index, semaphoreHolding := getSemaphoreHolding(wf.Status.Synchronization.Semaphore.Holding, lockKey)
			if index != -1 {
				semaphoreHolding.Holders = slice.RemoveString(semaphoreHolding.Holders, holdingName)
				wf.Status.Synchronization.Semaphore.Holding[index] = semaphoreHolding
			}
		}
	}
	return false
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

func getSemaphoreKey(namespace string, semaphoreRef *wfv1.SemaphoreRef) string {
	if semaphoreRef.ConfigMapKeyRef != nil {
		return fmt.Sprintf("%s/configmap/%s/%s", namespace, semaphoreRef.ConfigMapKeyRef.Name, semaphoreRef.ConfigMapKeyRef.Key)
	}
	return ""
}

func getResourceKey(namespace, wfName, resourceName string) string {
	resourceKey := fmt.Sprintf("%s/%s", namespace, wfName)
	// Template level TypeSemaphore
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
