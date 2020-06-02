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

	"github.com/argoproj/argo/errors"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/util/labels"
	"github.com/argoproj/argo/workflow/common"
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

func (cm *ConcurrencyManager) initialize(namespace string, wfClient wfclientset.Interface) error {
	labelSelector := v1Label.NewSelector()
	req, _ := v1Label.NewRequirement(common.LabelKeySemaphore, selection.Exists, nil)
	if req != nil {
		labelSelector.Add(*req)
	}
	req, _ = v1Label.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(wfv1.NodeRunning)})
	if req != nil {
		labelSelector = labelSelector.Add(*req)
	}

	listOpts := v1.ListOptions{LabelSelector: labelSelector.String()}
	wfList, err := wfClient.ArgoprojV1alpha1().Workflows(namespace).List(listOpts)

	if err != nil {
		return err
	}

	log.Infof("%d of workflows acquired the locks previously", wfList.Items.Len())

	for _, wf := range wfList.Items {
		annotation := wf.Annotations[common.AnnotationKeySemaphoreHolder]
		log.Infof("Annotation=%s", annotation)
		var semphoreMap map[string]interface{}
		err := json.Unmarshal([]byte(annotation), &semphoreMap)
		if err != nil {
			log.Errorf("%v", err)
		}
		log.Debugf("workflow %s and semaphore map %v, ", wf.Name, semphoreMap)
		for k, v := range semphoreMap {
			val := fmt.Sprintf("%s", v)
			var semaphore *Semaphore
			semaphore = cm.semaphoreMap[val]
			if semaphore == nil {
				semaphore, err = cm.initializeSemphore(val)
				if err != nil {
					return err
				}
				cm.semaphoreMap[val] = semaphore
			}
			if semaphore != nil && semaphore.acquire(k) {
				log.Infof("Lock Acquired by %s from %s", k, v)
			}
		}
	}
	log.Infof("ConcurrencyManager initialized successfully")
	return nil
}

func (cm *ConcurrencyManager) getConfigMapKeyRef(lockName string) (int, error) {
	items := strings.Split(lockName, "/")
	if len(items) < 4 {
		return 0, errors.New(errors.CodeBadRequest, "Invalid Config Map Key")
	}

	configMap, err := cm.controller.kubeclientset.CoreV1().ConfigMaps(items[0]).Get(items[2], v1.GetOptions{})

	if err != nil {
		return 0, err
	}

	value, ok := configMap.Data[items[3]]

	if !ok {
		return 0, errors.New(errors.CodeBadRequest, "Invalid Concurrency Key")
	}
	return strconv.Atoi(value)
}

func (cm *ConcurrencyManager) initializeSemphore(semphoreName string) (*Semaphore, error) {
	limit, err := cm.getConfigMapKeyRef(semphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semphoreName, limit, cm.controller), nil
}

func (cm *ConcurrencyManager) tryAcquire(key, namespace string, priority int32, creationTime time.Time, semaphoreRef *wfv1.SemaphoreRef, wf *wfv1.Workflow) (bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	lockName := getSemaphoreRefKey(namespace, semaphoreRef)
	var semaphore *Semaphore
	var err error
	semaphore, ok := cm.semaphoreMap[lockName]
	if !ok {
		if semaphoreRef.ConfigMapKeyRef != nil {
			semaphore, err = cm.initializeSemphore(lockName)
			if err != nil {
				return false, "", err
			}
			cm.semaphoreMap[lockName] = semaphore
		}
	}
	if semaphore == nil {
		return false, "", errors.New(errors.CodeBadRequest, "Requested Semaphore is invalid")
	}

	semaphore.addToQueue(key, priority, creationTime)
	status, msg := semaphore.tryAcquire(key)
	if status {
		cm.updateWorkflowMetaData(key, lockName, AcquireAction, wf)
	}
	return status, msg, nil
}

func (cm *ConcurrencyManager) release(key, namespace string, sem *wfv1.SemaphoreRef, wf *wfv1.Workflow) {
	if sem != nil {
		lockName := getSemaphoreRefKey(namespace, sem)
		if sem, ok := cm.semaphoreMap[lockName]; ok {
			sem.release(key)
			log.Debugf("%s semaphore lock is released by %s", lockName, key)
			cm.updateWorkflowMetaData(key, lockName, ReleaseAction, wf)
		}
	}
}

func (cm *ConcurrencyManager) releaseAll(wf *wfv1.Workflow) {
	if wf.Annotations == nil {
		return
	}
	semaphoreHolder := wf.Annotations[common.AnnotationKeySemaphoreHolder]
	if semaphoreHolder == "" {
		return
	}
	var holder map[string]interface{}
	err := json.Unmarshal([]byte(semaphoreHolder), &holder)
	if err != nil {
		log.Errorf("Invalid Semaphore Annotation. %v", err)
		return
	}
	for k, v := range holder {
		semaphoreName := fmt.Sprintf("%s", v)
		semaphore := cm.semaphoreMap[semaphoreName]
		if semaphore == nil {
			continue
		}
		semaphore.release(k)
	}
}

func (cm *ConcurrencyManager) getHolderKey(wf *wfv1.Workflow, nodeName string) string {
	if wf == nil {
		return ""
	}
	key := fmt.Sprintf("%s/%s", wf.Namespace, wf.Name)
	if nodeName != "" {
		key = fmt.Sprintf("%s/%s", key, nodeName)
	}
	return key
}

func (cm *ConcurrencyManager) updateWorkflowMetaData(key, semaphoreKey, action string, wf *wfv1.Workflow) {

	labels.Label(wf, common.LabelKeySemaphore, "true")

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
			log.Errorf("Invalid Semaphore Annotation. %v", err)
			return
		}
	}
	if action == AcquireAction {
		holder[key] = semaphoreKey
	}
	if action == ReleaseAction {
		log.Debugf("%s removed from Annotation", key)
		delete(holder, key)
	}
	holderStr, err := json.Marshal(holder)
	if err != nil {
		log.Errorf("Invalid Semaphore Annotation. %v", err)
		return
	}
	wf.Annotations[common.AnnotationKeySemaphoreHolder] = string(holderStr)
}

func getSemaphoreRefKey(namespace string, sem *wfv1.SemaphoreRef) string {
	key := namespace
	if sem.ConfigMapKeyRef != nil {
		key = fmt.Sprintf("%s/%s/%s/%s", namespace, "configmap", sem.ConfigMapKeyRef.Name, sem.ConfigMapKeyRef.Key)
	}
	return key
}
