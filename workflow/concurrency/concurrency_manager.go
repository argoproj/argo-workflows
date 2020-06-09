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

type ConcurrencyManager struct {
	kubeClient        kubernetes.Interface
	concurrencyMap    map[string]Concurrency
	lock              *sync.Mutex
	releaseNotifyFunc ReleaseNotifyCallbackFunc
}

type Concurrency interface {
	acquire(holderKey string) LockStatus
	tryAcquire(holderKey string) (LockStatus, string)
	release(key string) LockStatus
	addToQueue(holderKey string, priority int32, creationTime time.Time)
	getCurrentHolders() []string
	getName() string
	getLimit() int
	resize(n int) bool
}

type LockStatus string

const (
	Acquired        LockStatus = "acquired"
	AlreadyAcquired LockStatus = "alreadyAcquired"
	Released        LockStatus = "released"
	Waiting         LockStatus = "waiting"
)

func NewConcurrencyManager(kubeClient kubernetes.Interface, callbackFunc func(string)) *ConcurrencyManager {
	return &ConcurrencyManager{
		kubeClient:        kubeClient,
		concurrencyMap:    make(map[string]Concurrency),
		lock:              &sync.Mutex{},
		releaseNotifyFunc: callbackFunc,
	}
}

func (cm *ConcurrencyManager) Initialize(namespace string, wfClient wfclientset.Interface) {
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
					log.Warnf("Concurrency configmap %s is not found. %v", v, err)
					continue
				}

				cm.concurrencyMap[k] = semaphore
			}
			for _, ele := range v.Name {

				resourceKey := cm.getResourceKey(wf.Namespace, wf.Name, ele)
				if semaphore != nil && semaphore.acquire(resourceKey) == Acquired {
					log.Infof("Lock Acquired by %s from %s", resourceKey, k)
				}
			}
		}
	}
	log.Infof("ConcurrencyManager initialized successfully")
}

func (cm *ConcurrencyManager) getCurrentLockHolders(lockName string) []string {
	if concurrency, ok := cm.concurrencyMap[lockName]; ok {
		return concurrency.getCurrentHolders()
	}
	return nil
}

func (cm *ConcurrencyManager) getResourceKey(namespace, wfName, resourceName string) string {
	resourceKey := fmt.Sprintf("%s/%s", namespace, wfName)
	// Template level Semaphore
	if resourceName != wfName {
		resourceKey = fmt.Sprintf("%s/%s", resourceKey, resourceName)
	}
	return resourceKey
}

func (cm *ConcurrencyManager) getConfigMapKeyRef(lockName string) (int, error) {
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
		return 0, errors.New(errors.CodeBadRequest, "Invalid Concurrency Key")
	}
	return strconv.Atoi(value)
}

func (cm *ConcurrencyManager) initializeSemaphore(semaphoreName string) (Concurrency, error) {
	limit, err := cm.getConfigMapKeyRef(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.releaseNotifyFunc), nil
}

func (cm *ConcurrencyManager) isSemaphoreSizeChanged(semaphore Concurrency) (bool, int, error) {
	limit, err := cm.getConfigMapKeyRef(semaphore.getName())
	if err != nil {
		return false, semaphore.getLimit(), err
	}
	return !(semaphore.getLimit() == limit), limit, nil
}

func (cm *ConcurrencyManager) checkAndUpdateSemaphoreSize(semaphore Concurrency) error {

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
func (cm *ConcurrencyManager) TryAcquire(key, namespace string, priority int32, creationTime time.Time, concurrencyRef wfv1.ConcurrencyRef, wf *wfv1.Workflow) (bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	lockName := concurrencyRef.GetKey(namespace)
	var concurrency Concurrency
	var err error

	concurrency, found := cm.concurrencyMap[lockName]

	if !found {
		if concurrencyRef.GetType() == wfv1.Semaphore {
			concurrency, err = cm.initializeSemaphore(lockName)
			if err != nil {
				return false, "", err
			}
			cm.concurrencyMap[lockName] = concurrency
		}
	}

	if concurrency == nil {
		return false, "", errors.New(errors.CodeBadRequest, "Requested Concurrency is invalid")
	}

	// Check concurrency configmap changes
	err = cm.checkAndUpdateSemaphoreSize(concurrency)

	if err != nil {
		return false, "", err
	}

	concurrency.addToQueue(key, priority, creationTime)

	status, msg := concurrency.tryAcquire(key)
	if status == AlreadyAcquired {
		return true, "", nil
	} else if status == Acquired {
		cm.updateWorkflowMetaData(key, lockName, concurrencyRef.GetType(), status, wf)
		return true, "", nil
	}

	cm.updateWorkflowMetaData(key, lockName, concurrencyRef.GetType(), status, wf)
	return false, msg, nil
}

func (cm *ConcurrencyManager) Release(key, namespace string, concurrencyRef wfv1.ConcurrencyRef, wf *wfv1.Workflow) {
	lockName := concurrencyRef.GetKey(namespace)
	if concurrency, ok := cm.concurrencyMap[lockName]; ok {
		concurrency.release(key)
		log.Debugf("%s concurrecy lock is released by %s", lockName, key)
		cm.updateWorkflowMetaData(key, lockName, concurrencyRef.GetType(), Released, wf)
	}
}

func (cm *ConcurrencyManager) ReleaseAll(wf *wfv1.Workflow) bool {
	released := false
	if wf.Spec.Semaphore != nil {

		if wf.Status.Concurrency == nil || wf.Status.Concurrency.Semaphore == nil {
			return released
		}
		for k, v := range wf.Status.Concurrency.Semaphore.Holding {
			concurrency := cm.concurrencyMap[k]
			if concurrency == nil {
				continue
			}
			for _, ele := range v.Name {
				resourceKey := cm.getResourceKey(wf.Namespace, wf.Name, ele)
				concurrency.release(resourceKey)
				cm.updateWorkflowMetaData(ele, k, wfv1.Semaphore, Released, wf)
				released = true
				log.Infof("%s released a lock from %s", resourceKey, k)
			}
		}
		// Clear the Concurrency details
		wf.Status.Concurrency = nil
	}
	return released
}

func (cm *ConcurrencyManager) GetHolderKey(wf *wfv1.Workflow, nodeName string) string {
	if wf == nil {
		return ""
	}
	key := fmt.Sprintf("%s/%s", wf.Namespace, wf.Name)
	if nodeName != "" {
		key = fmt.Sprintf("%s/%s", key, nodeName)
	}
	return key
}

func (cm *ConcurrencyManager) updateWorkflowMetaData(key, semaphoreKey string, concurrencyType wfv1.ConcurrencyType, action LockStatus, wf *wfv1.Workflow) {

	if wf.Status.Concurrency == nil {
		wf.Status.Concurrency = &wfv1.ConcurrencyStatus{Semaphore: &wfv1.SemaphoreStatus{}}
	}
	if concurrencyType == wfv1.Semaphore {
		if wf.Status.Concurrency.Semaphore == nil {
			wf.Status.Concurrency.Semaphore = &wfv1.SemaphoreStatus{}
		}

		if action == Waiting {
			if wf.Status.Concurrency.Semaphore.Waiting == nil {
				wf.Status.Concurrency.Semaphore.Waiting = make(map[string]wfv1.WaitingStatus)
			}
			wf.Status.Concurrency.Semaphore.Waiting[semaphoreKey] = wfv1.WaitingStatus{Holders: wfv1.HolderNames{Name: cm.getCurrentLockHolders(semaphoreKey),},}

			return
		}

		if action == Acquired {
			if wf.Status.Concurrency.Semaphore.Holding == nil {
				wf.Status.Concurrency.Semaphore.Holding = make(map[string]wfv1.HolderNames)
			}
			holding := wf.Status.Concurrency.Semaphore.Holding[semaphoreKey]
			if holding.Name == nil {
				holding = wfv1.HolderNames{}
			}
			items := strings.Split(key, "/")

			holding.Name = append(holding.Name, items[len(items)-1])
			wf.Status.Concurrency.Semaphore.Holding[semaphoreKey] = holding
			return
		}
		if action == Released {
			log.Debugf("%s removed from Status", key)
			holding := wf.Status.Concurrency.Semaphore.Holding[semaphoreKey]
			if holding.Name != nil {
				holding.Name = RemoveFromSlice(holding.Name, key)
			}
			if len(holding.Name) == 0 {
				delete(wf.Status.Concurrency.Semaphore.Holding, semaphoreKey)
			}else {
				wf.Status.Concurrency.Semaphore.Holding[semaphoreKey] = holding
			}
			return
		}
	}
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
