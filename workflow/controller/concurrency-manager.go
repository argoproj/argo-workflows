package controller

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/argoproj/argo/errors"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

type ConcurrencyManager struct {
	semaphoreMap map[string]*Semaphore
	lock         *sync.Mutex
	controller   *WorkflowController
}

func NewConcurrencyManager(controller *WorkflowController) *ConcurrencyManager {
	return &ConcurrencyManager{
		semaphoreMap: make(map[string]*Semaphore),
		lock:         &sync.Mutex{},
		controller:   controller,
	}
}

func (wt *ConcurrencyManager) GetConfigMapKeyRef(lockName string) (int, error) {
	items := strings.Split(lockName, "/")
	if len(items) < 4 {
		return 0, errors.New(errors.CodeBadRequest, "Invalid Config Map Key")
	}

	configMap, err := wt.controller.kubeclientset.CoreV1().ConfigMaps(items[0]).Get(items[2], v1.GetOptions{})

	if err != nil {
		return 0, err
	}

	value, ok := configMap.Data[items[3]]

	if !ok {
		return 0, errors.New(errors.CodeBadRequest, "Invalid Concurrency Key")
	}
	return strconv.Atoi(value)
}

func getSemaphoreRefKey(namespace string, sem *wfv1.SemaphoreRef) string {
	key := namespace
	if sem.ConfigMapKeyRef != nil {
		key = fmt.Sprintf("%s/%s/%s/%s", namespace, "configmap", sem.ConfigMapKeyRef.Name, sem.ConfigMapKeyRef.Key)
	}
	return key
}

func (wt *ConcurrencyManager) TryAcquire(key, namespace string, priority int32, creationTime time.Time, semaphoreRef *wfv1.SemaphoreRef) (bool, string, error) {
	wt.lock.Lock()
	defer wt.lock.Unlock()
	lockName := getSemaphoreRefKey(namespace, semaphoreRef)
	sema, ok := wt.semaphoreMap[lockName]

	if !ok {
		if semaphoreRef.ConfigMapKeyRef != nil {
			limit, err := wt.GetConfigMapKeyRef(lockName)
			if err != nil {
				return false, "", err
			}
			sema = NewSemaphore(lockName, limit, wt.controller)
			wt.semaphoreMap[lockName] = sema
		}
	}
	if sema == nil {
		return false, "", errors.New(errors.CodeBadRequest, "Requested Semaphore is invalid")
	}

	sema.AddToQueue(key, priority, creationTime)
	status, msg := sema.TryAcquire(key)
	return status, msg, nil
}

func (wt *ConcurrencyManager) Release(key, namespace string, sem *wfv1.SemaphoreRef) {
	if sem != nil {
		semaKey := getSemaphoreRefKey(namespace, sem)
		if sem, ok := wt.semaphoreMap[semaKey]; ok {
			sem.Release(key)
		}
	}
}

func (wt *ConcurrencyManager) getHolderKey(wf *wfv1.Workflow, nodeName string) string {
	if wf == nil {
		return ""
	}
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err != nil {
		return ""
	}
	if nodeName != "" {
		key = fmt.Sprintf("%s/%s", key, nodeName)
	}
	return key
}

