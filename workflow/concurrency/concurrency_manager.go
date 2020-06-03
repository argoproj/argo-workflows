package concurrency

import (
	"encoding/json"
	"fmt"
	"k8s.io/client-go/kubernetes"
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

type ReleaseNotifyCallbackFunc func(string)

type ConcurrencyManager struct {
	kubeClient        kubernetes.Interface
	semaphoreMap      map[string]*Semaphore
	lock              *sync.Mutex
	releaseNotifyFunc ReleaseNotifyCallbackFunc
}

const (
	AcquireAction = "acquired"
	ReleaseAction = "released"
)

func NewConcurrencyManager(kubeClient kubernetes.Interface, callbackFunc func(string)) *ConcurrencyManager {
	return &ConcurrencyManager{
		kubeClient:        kubeClient,
		semaphoreMap:      make(map[string]*Semaphore),
		lock:              &sync.Mutex{},
		releaseNotifyFunc: callbackFunc,
	}
}

func (cm *ConcurrencyManager) Initialize(namespace string, wfClient wfclientset.Interface) error {
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
		var semaphoreMap map[string]interface{}
		err := json.Unmarshal([]byte(annotation), &semaphoreMap)
		if err != nil {
			log.Errorf("%v", err)
		}
		log.Debugf("workflow %s and semaphore map %v, ", wf.Name, semaphoreMap)
		for k, v := range semaphoreMap {
			val := fmt.Sprintf("%s", v)
			var semaphore *Semaphore
			semaphore = cm.semaphoreMap[val]
			if semaphore == nil {
				semaphore, err = cm.initializeSemaphore(val)
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

func (cm *ConcurrencyManager) initializeSemaphore(semaphoreName string) (*Semaphore, error) {
	limit, err := cm.getConfigMapKeyRef(semaphoreName)
	if err != nil {
		return nil, err
	}
	return NewSemaphore(semaphoreName, limit, cm.releaseNotifyFunc), nil
}

func (cm *ConcurrencyManager) isSemaphoreSizeChanged(semaphore *Semaphore) (bool, int, error) {
	limit, err := cm.getConfigMapKeyRef(semaphore.name)
	if err != nil {
		return false, semaphore.limit, err
	}
	return !(semaphore.limit == limit), limit, nil
}

func (cm *ConcurrencyManager) checkAndUpdateSemaphoreSize(semaphore *Semaphore) (bool, error) {

	changed,newLimit, err := cm.isSemaphoreSizeChanged(semaphore)
	if err != nil {
		return false, err
	}
	if changed {
		return semaphore.resize(newLimit) , nil
	}
	return changed, nil
}

// TryAcquire tries to acquire the lock from semaphore.
// It returns status of lock acquire, waiting message if lock is not available and any error encountered
func (cm *ConcurrencyManager) TryAcquire(key, namespace string, priority int32, creationTime time.Time, semaphoreRef *wfv1.SemaphoreRef, wf *wfv1.Workflow) (bool, string, error) {
	cm.lock.Lock()
	defer cm.lock.Unlock()
	lockName := getSemaphoreRefKey(namespace, semaphoreRef)
	var semaphore *Semaphore
	var err error
	semaphore, ok := cm.semaphoreMap[lockName]
	if !ok {
		if semaphoreRef.ConfigMapKeyRef != nil {
			semaphore, err = cm.initializeSemaphore(lockName)
			if err != nil {
				return false, "", err
			}
			cm.semaphoreMap[lockName] = semaphore
		}
	}
	if semaphore == nil {
		return false, "", errors.New(errors.CodeBadRequest, "Requested Semaphore is invalid")
	}

	_, err = cm.checkAndUpdateSemaphoreSize(semaphore)

	if err != nil {
		return false, "", err
	}

	semaphore.addToQueue(key, priority, creationTime)
	status, msg := semaphore.tryAcquire(key)
	if status {
		cm.updateWorkflowMetaData(key, lockName, AcquireAction, wf)
	}
	return status, msg, nil
}

func (cm *ConcurrencyManager) Release(key, namespace string, sem *wfv1.SemaphoreRef, wf *wfv1.Workflow) {
	if sem != nil {
		lockName := getSemaphoreRefKey(namespace, sem)
		if sem, ok := cm.semaphoreMap[lockName]; ok {
			sem.release(key)
			log.Debugf("%s semaphore lock is released by %s", lockName, key)
			cm.updateWorkflowMetaData(key, lockName, ReleaseAction, wf)
		}
	}
}

func (cm *ConcurrencyManager) ReleaseAll(wf *wfv1.Workflow) {
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
