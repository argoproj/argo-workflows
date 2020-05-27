package controller

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/util/labels"
)

type ConcurrencyManager struct {
	semaphoreMap map[string]*Semaphore
	lock         *sync.Mutex
	controller   *WorkflowController
}

const (
	AcquireAction = "acquired"
	ReleaseAction = "released"
)

func NewConcurrencyManager(controller *WorkflowController) *ConcurrencyManager {
	return &ConcurrencyManager{
		semaphoreMap: make(map[string]*Semaphore),
		lock:         &sync.Mutex{},
		controller:   controller,
	}
}

func (wt *ConcurrencyManager) initialize(namespace string, wfClient wfclientset.Interface) error {
	labelSelector := v1Label.NewSelector()
	req, _ := v1Label.NewRequirement(common.LabelKeySemaphore, selection.Exists, nil)
	if req != nil {
		labelSelector.Add(*req)
	}
	req, _ = v1Label.NewRequirement(common.LabelKeyPhase, selection.In, []string{"Running"})
	if req != nil {
		labelSelector = labelSelector.Add(*req)
	}

	listOpts := v1.ListOptions{LabelSelector: labelSelector.String()}
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(listOpts)

	if err != nil {
		return err
	}

	log.Infof("%d of Workflow previously acquired the locks", wfList.Items.Len())

	for _, wf := range wfList.Items {
		annotation := wf.Annotations[common.LabelKeySemaphore]
		var semphoreMap map[string]interface{}
		err := json.Unmarshal([]byte(annotation), &semphoreMap)
		if err != nil {
			log.Errorf("%v", err)
		}
		log.Debugf("workflow %s and semaphore map %v, ", wf.Name, semphoreMap)
		for k, v := range semphoreMap {
			val := fmt.Sprintf("%s", v)
			var sema *Semaphore
			sema = wt.semaphoreMap[val]
			if sema == nil {
				sema, err = wt.initializeSemphore(val)
				if err != nil {
					return err
				}
				wt.semaphoreMap[val] = sema
			}
			if sema != nil && sema.Acquire(k) {
				log.Debugf("Lock Acquired by %s from %s", k, v)
			}
		}
	}
	return nil
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

func (wt *ConcurrencyManager) initializeSemphore(semphoreName string) (*Semaphore, error) {
	limit, err := wt.GetConfigMapKeyRef(semphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semphoreName, limit, wt.controller), nil
}

func (wt *ConcurrencyManager) TryAcquire(key, namespace string, priority int32, creationTime time.Time, semaphoreRef *wfv1.SemaphoreRef, wf *wfv1.Workflow) (bool, string, error) {
	wt.lock.Lock()
	defer wt.lock.Unlock()
	lockName := getSemaphoreRefKey(namespace, semaphoreRef)
	var sema *Semaphore
	var err error
	sema, ok := wt.semaphoreMap[lockName]
	if !ok {
		if semaphoreRef.ConfigMapKeyRef != nil {
			sema, err = wt.initializeSemphore(lockName)
			if err != nil {
				return false, "", err
			}
			wt.semaphoreMap[lockName] = sema
		}
	}
	if sema == nil {
		return false, "", errors.New(errors.CodeBadRequest, "Requested Semaphore is invalid")
	}

	sema.AddToQueue(key, priority, creationTime)
	status, msg := sema.TryAcquire(key)
	if status {
		wt.updateWorkflowMetaData(key, lockName, AcquireAction, wf)
	}
	return status, msg, nil
}

func (wt *ConcurrencyManager) Release(key, namespace string, sem *wfv1.SemaphoreRef, wf *wfv1.Workflow) {
	if sem != nil {
		lockName := getSemaphoreRefKey(namespace, sem)
		if sem, ok := wt.semaphoreMap[lockName]; ok {
			sem.Release(key)
			wt.updateWorkflowMetaData(key, lockName, ReleaseAction, wf)
		}
	}
}

func (wt *ConcurrencyManager) getHolderKey(wf *wfv1.Workflow, nodeName string) (string, error) {
	if wf == nil {
		return "", errors.Errorf(errors.CodeBadRequest, "Invalid Workflow object")
	}
	key, err := cache.MetaNamespaceKeyFunc(wf)
	if err != nil {
		return "", err
	}
	if nodeName != "" {
		key = fmt.Sprintf("%s/%s", key, nodeName)
	}
	return key, nil
}

func (wt *ConcurrencyManager) updateWorkflowMetaData(key, semaphoreKey, action string, wf *wfv1.Workflow) {

	labels.Label(wf,common.LabelKeySemaphore, "true")

	if wf.Annotations == nil {
		wf.Annotations = make(map[string]string)
	}
	var holder map[string]interface{}
	semaphoreHolder := wf.Annotations[common.AnnotationKeySemaphoreHolder]
	if semaphoreHolder == "" {
		holder = make(map[string]interface{})
	} else {
		err := json.Unmarshal([]byte(semaphoreHolder), &holder)
		if err != nil {

		}
	}
	if action == AcquireAction {
		holder[key] = semaphoreKey
	}
	if action == ReleaseAction {
		log.Infof("Removed from Annotation %k", key)
		delete(holder, key)
		log.Infof("%v", holder)
	}
	holderStr, err := json.Marshal(holder)
	if err != nil {

	}
	wf.Annotations[common.AnnotationKeySemaphoreHolder] = string(holderStr)
}
