package ttlcontroller

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/clock"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

const (
	workflowTTLResyncPeriod = 20 * time.Minute
)

type Controller struct {
	wfclientset  wfclientset.Interface
	wfInformer   cache.SharedIndexInformer
	workqueue    workqueue.DelayingInterface
	resyncPeriod time.Duration
	clock        clock.Clock
}

// NewController returns a new workflow ttl controller
func NewController(config *rest.Config, wfClientset wfclientset.Interface, namespace, instanceID string) *Controller {
	filterCompletedWithTTL := func(options *metav1.ListOptions) {
		// completed equals (true)
		completedReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"true"})
		if err != nil {
			panic(err)
		}
		labelSelector := labels.NewSelector().
			Add(*completedReq).
			Add(util.InstanceIDRequirement(instanceID))
		options.LabelSelector = labelSelector.String()
	}
	wfInformer := util.NewWorkflowInformer(config, namespace, workflowTTLResyncPeriod, filterCompletedWithTTL)

	controller := &Controller{
		wfclientset:  wfClientset,
		wfInformer:   wfInformer,
		workqueue:    workqueue.NewNamedDelayingQueue("workflow-ttl"),
		resyncPeriod: workflowTTLResyncPeriod,
		clock:        clock.RealClock{},
	}

	wfInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueWF,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueWF(new)
		},
		DeleteFunc: controller.enqueueWF,
	})
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}) error {
	defer runtimeutil.HandleCrash()
	defer c.workqueue.ShutDown()
	log.Infof("Starting workflow TTL controller (resync %v)", c.resyncPeriod)
	go c.wfInformer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.wfInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	go wait.Until(c.runWorker, time.Second, stopCh)
	log.Info("Started workflow TTL worker")
	<-stopCh
	log.Info("Shutting workflow TTL worker")
	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			//c.workqueue.Forget(obj)
			runtimeutil.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.deleteWorkflow(key); err != nil {
			return fmt.Errorf("error deleting '%s': %s", key, err.Error())
		}
		//c.workqueue.Forget(obj)
		return nil
	}(obj)

	if err != nil {
		runtimeutil.HandleError(err)
		return true
	}

	return true
}

// enqueueWF conditionally queues a workflow to the ttl queue if it is within the deletion period
func (c *Controller) enqueueWF(obj interface{}) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("'%v' is not an unstructured", obj)
		return
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.Warnf("Failed to unmarshal workflow %v object: %v", obj, err)
		return
	}
	now := c.clock.Now()
	remaining, expiration := timeLeft(wf, &now)
	if remaining == nil || *remaining > c.resyncPeriod {
		return
	}
	log.Infof("Found Workflow %s/%s set expire at %v (%s from now)", wf.Namespace, wf.Name, expiration, remaining)
	var addAfter time.Duration
	if *remaining > 0 {
		addAfter = *remaining
	}
	var key string
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtimeutil.HandleError(err)
		return
	}
	//c.workqueue.Add(key)
	log.Infof("Queueing workflow %s/%s for delete in %v", wf.Namespace, wf.Name, addAfter)
	c.workqueue.AddAfter(key, addAfter)
}

func (c *Controller) deleteWorkflow(key string) error {
	obj, exists, err := c.wfInformer.GetIndexer().GetByKey(key)
	if err != nil {
		if apierr.IsNotFound(err) {
			runtimeutil.HandleError(fmt.Errorf("foo '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}
	if !exists {
		return nil
	}

	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Key '%s' in index is not an unstructured", key)
		return nil
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.Warnf("Failed to unmarshal key '%s' to workflow object: %v", key, err)
		return nil
	}
	if c.ttlExpired(wf) {
		log.Infof("Deleting TTL expired workflow %s/%s", wf.Namespace, wf.Name)
		policy := metav1.DeletePropagationForeground
		err = c.wfclientset.Argoproj().Workflows(wf.Namespace).Delete(wf.Name, &metav1.DeleteOptions{PropagationPolicy: &policy})
		if err != nil {
			return err
		}
		log.Infof("Successfully deleted '%s'", key)
	}
	return nil
}

func (c *Controller) ttlExpired(wf *wfv1.Workflow) bool {
	// We don't care about the Workflows that are going to be deleted, or the ones that don't need clean up.
	if wf.DeletionTimestamp != nil || wf.Spec.TTLSecondsAfterFinished == nil || wf.Status.FinishedAt.IsZero() {
		return false
	}
	now := c.clock.Now()
	expiry := wf.Status.FinishedAt.Add(time.Second * time.Duration(*wf.Spec.TTLSecondsAfterFinished))
	return now.After(expiry)
}

func timeLeft(wf *wfv1.Workflow, since *time.Time) (*time.Duration, *time.Time) {
	if wf.DeletionTimestamp != nil || wf.Spec.TTLSecondsAfterFinished == nil || wf.Status.FinishedAt.IsZero() {
		return nil, nil
	}
	sinceUTC := since.UTC()
	finishAtUTC := wf.Status.FinishedAt.UTC()
	if finishAtUTC.After(sinceUTC) {
		log.Infof("Warning: Found Workflow %s/%s finished in the future. This is likely due to time skew in the cluster. Workflow cleanup will be deferred.", wf.Namespace, wf.Name)
	}
	expireAtUTC := finishAtUTC.Add(time.Duration(*wf.Spec.TTLSecondsAfterFinished) * time.Second)
	remaining := expireAtUTC.Sub(sinceUTC)
	return &remaining, &expireAtUTC
}
