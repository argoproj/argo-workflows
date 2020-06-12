package concurrency

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
)

type ReleaseNotifyCallbackFunc func(string)

type LockManager struct {
	kubeClient        kubernetes.Interface
	concurrencyMap    map[string]Concurrency
	lock              *sync.Mutex
	releaseNotifyFunc ReleaseNotifyCallbackFunc
}

type Concurrency interface {
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

func NewLockManager(kubeClient kubernetes.Interface, callbackFunc func(string)) *LockManager {
	return &LockManager{
		kubeClient:        kubeClient,
		concurrencyMap:    make(map[string]Concurrency),
		lock:              &sync.Mutex{},
		releaseNotifyFunc: callbackFunc,
	}
}

func (cm *LockManager) Initialize(namespace string, wfClient wfclientset.Interface) {
	labelSelector := v1Label.NewSelector()
	req, _ := v1Label.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(wfv1.NodeRunning)})
	if req != nil {
		labelSelector = labelSelector.Add(*req)
	}
	listOpts := v1.ListOptions{LabelSelector: labelSelector.String()}
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(listOpts)

	if err != nil {
		log.Warnf("List workflow operation failed. %v", err)
		return
	}

	for _, wf := range wfList.Items {
		if wf.Status.Concurrency == nil || wf.Status.Concurrency.Semaphore == nil || wf.Status.Concurrency.Semaphore.Holding == nil {
			continue
		}
		for k, v := range wf.Status.Concurrency.Semaphore.Holding {
			var semaphore Concurrency
			semaphore = cm.concurrencyMap[k]
			if semaphore == nil {
				semaphore, err = cm.initializeSemaphore(k)
				if err != nil {
					log.Warnf("ConcurrencyRef configmap %s is not found. %v", v, err)
					continue
				}

				cm.concurrencyMap[k] = semaphore
			}
			for _, ele := range v.Name {

				resourceKey := getResourceKey(wf.Namespace, wf.Name, ele)
				if semaphore != nil && semaphore.acquire(resourceKey) {
					log.Infof("Lock acquired by %s from %s", resourceKey, k)
				}
			}
		}
	}
	log.Infof("LockManager initialized successfully")
}

func (cm *LockManager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := cm.concurrencyMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (cm *LockManager) getConfigMapKeyRef(lockName string) (int, error) {
	items := strings.Split(lockName, "/")
	if len(items) < 4 {
		return 0, errors.New(errors.CodeBadRequest, "Invalid Config Map Key")
	}

	configMap, err := cm.kubeClient.CoreV1().ConfigMaps(items[0]).Get(items[2], v1.GetOptions{})

	if err != nil {
		return 0, err
	}

	value, ok := configMap.Data[items[3]]

	if !ok {
		return 0, errors.New(errors.CodeBadRequest, "Invalid ConcurrencyRef Key")
	}
	return strconv.Atoi(value)
}

func (cm *LockManager) initializeSemaphore(semaphoreName string) (Concurrency, error) {
	limit, err := cm.getConfigMapKeyRef(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.releaseNotifyFunc), nil
}

func (cm *LockManager) isSemaphoreSizeChanged(semaphore Concurrency) (bool, int, error) {
	limit, err := cm.getConfigMapKeyRef(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return !(semaphore.getLimit() == limit), limit, nil
}

func (cm *LockManager) checkAndUpdateSemaphoreSize(semaphore Concurrency) error {

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
// It returns status of acquiring a lock , waiting message if lock is not available and any error encountered
func (cm *LockManager) TryAcquire(wf *wfv1.Workflow, nodeName string, priority int32, creationTime time.Time, concurrencyRef *wfv1.ConcurrencyRef) (bool, bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if concurrencyRef == nil {
		return true, false, "", nil
	}

	var lock Concurrency
	var lockKey string
	var lockType LockType
	var err error

	if concurrencyRef.Semaphore != nil {
		var found bool
		lockKey = getSemaphoreKey(wf.Namespace, concurrencyRef.Semaphore)

		lock, found = cm.concurrencyMap[lockKey]

		if !found {
			lock, err = cm.initializeSemaphore(lockKey)
			if err != nil {
				return false, false, "", err
			}
			cm.concurrencyMap[lockKey] = lock
		}

		lock, found = cm.concurrencyMap[lockKey]
		if !found {
			return false, false, "", errors.New(errors.CodeBadRequest, "Requested SemaphoreRef is invalid")
		}

		// Check lock configmap changes
		err := cm.checkAndUpdateSemaphoreSize(lock)

		if err != nil {
			return false, false, "", err
		}
		lockType = TypeSemaphore
	}

	if lockKey == "" {
		return false, false, "", errors.New(errors.CodeBadRequest, "Requested Concurrency is invalid")
	}

	holderKey := getHolderKey(wf, nodeName)

	lock.addToQueue(holderKey, priority, creationTime)

	status, msg := lock.tryAcquire(holderKey)
	if status {
		updated := cm.updateConcurrencyStatus(holderKey, lockKey, lockType, acquired, wf)
		return true, updated, "", nil
	}

	udpated := cm.updateConcurrencyStatus(holderKey, lockKey, lockType, waiting, wf)
	return false, udpated, msg, nil
}

func (cm *LockManager) Release(wf *wfv1.Workflow, nodeName, namespace string, concurrencyRef *wfv1.ConcurrencyRef) {
	if concurrencyRef == nil {
		return
	}
	holderKey := getHolderKey(wf, nodeName)
	if concurrencyRef.Semaphore != nil {
		concurrencyKey := getSemaphoreKey(namespace, concurrencyRef.Semaphore)
		if concurrency, ok := cm.concurrencyMap[concurrencyKey]; ok {
			concurrency.release(holderKey)
			log.Debugf("%s concurrency lock is released by %s", concurrencyKey, holderKey)
			cm.updateConcurrencyStatus(holderKey, concurrencyKey, TypeSemaphore, released, wf)
		}
	}
}

func (cm *LockManager) ReleaseAll(wf *wfv1.Workflow) bool {
	if wf.Status.Concurrency == nil {
		return true
	}
	if wf.Status.Concurrency.Semaphore != nil {
		for k, v := range wf.Status.Concurrency.Semaphore.Holding {
			concurrency := cm.concurrencyMap[k]
			if concurrency == nil {
				continue
			}
			for _, ele := range v.Name {
				resourceKey := getResourceKey(wf.Namespace, wf.Name, ele)
				concurrency.release(resourceKey)
				cm.updateConcurrencyStatus(ele, k, TypeSemaphore, released, wf)
				log.Infof("%s released a lock from %s", resourceKey, k)
			}
		}
		// Clear the ConcurrencyRef details
		wf.Status.Concurrency.Semaphore = nil
		wf.Status.Concurrency = nil
	}
	return true
}

func (cm *LockManager) updateConcurrencyStatus(holderKey, lockKey string, lockType LockType, lockAction LockAction, wf *wfv1.Workflow) bool {

	if wf.Status.Concurrency == nil {
		wf.Status.Concurrency = &wfv1.ConcurrencyStatus{Semaphore: &wfv1.SemaphoreStatus{}}
	}
	if lockType == TypeSemaphore {
		if wf.Status.Concurrency.Semaphore == nil {
			wf.Status.Concurrency.Semaphore = &wfv1.SemaphoreStatus{}
		}

		if lockAction == waiting {
			if wf.Status.Concurrency.Semaphore.Waiting == nil {
				wf.Status.Concurrency.Semaphore.Waiting = make(map[string]wfv1.WaitingStatus)
			}
			wf.Status.Concurrency.Semaphore.Waiting[lockKey] = wfv1.WaitingStatus{Holders: wfv1.HolderNames{Name: cm.getCurrentLockHolders(lockKey)}}
			return true
		}

		if lockAction == acquired {
			if wf.Status.Concurrency.Semaphore.Holding == nil {
				wf.Status.Concurrency.Semaphore.Holding = make(map[string]wfv1.HolderNames)
			}
			holding := wf.Status.Concurrency.Semaphore.Holding[lockKey]
			if holding.Name == nil {
				holding = wfv1.HolderNames{}
			}
			items := strings.Split(holderKey, "/")
			holdingName := items[len(items)-1]
			if !Contains(holding.Name, holdingName) {
				holding.Name = append(holding.Name, items[len(items)-1])
				wf.Status.Concurrency.Semaphore.Holding[lockKey] = holding
				return true
			}
			return false
		}
		if lockAction == released {
			log.Debugf("%s removed from Status", holderKey)
			holding := wf.Status.Concurrency.Semaphore.Holding[lockKey]
			if holding.Name != nil {
				holding.Name = RemoveFromSlice(holding.Name, holderKey)
			}
			if len(holding.Name) == 0 {
				delete(wf.Status.Concurrency.Semaphore.Holding, lockKey)
			} else {
				wf.Status.Concurrency.Semaphore.Holding[lockKey] = holding
			}
			return true
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

// TODO -- Need to move it to util package -Bala
func RemoveFromSlice(slice []string, element string) []string {
	n := len(slice)
	if n == 1 {
		return []string{}
	}
	for i, v := range slice {
		if element == v {
			if n-2 < i {
				slice = append(slice[:i], slice[i+1:]...)
			} else {
				slice = slice[:i]
			}
		}
	}
	return slice
}

// TODO -- Need to move it to util package -Bala
func Contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
