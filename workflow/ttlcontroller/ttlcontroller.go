package ttlcontroller

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/clock"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	commonutil "github.com/argoproj/argo/util"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/util"
)

type Controller struct {
	wfclientset wfclientset.Interface
	wfInformer  cache.SharedIndexInformer
	workqueue   workqueue.DelayingInterface
	clock       clock.Clock
}

// NewController returns a new workflow ttl controller
func NewController(wfClientset wfclientset.Interface, wfInformer cache.SharedIndexInformer) *Controller {
	controller := &Controller{
		wfclientset: wfClientset,
		wfInformer:  wfInformer,
		workqueue:   workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow_ttl_queue"),
		clock:       clock.RealClock{},
	}

	wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			return ok && un.GetDeletionTimestamp() == nil && un.GetLabels()[common.LabelKeyCompleted] == "true" && un.GetLabels()[common.LabelKeyWorkflowArchivingStatus] != "Pending"
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: controller.enqueueWF,
			UpdateFunc: func(old, new interface{}) {
				controller.enqueueWF(new)
			},
		},
	})
	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}, workflowTTLWorkers int) error {
	defer runtimeutil.HandleCrash()
	defer c.workqueue.ShutDown()
	log.Infof("Starting workflow TTL controller (workflowTTLWorkers %d)", workflowTTLWorkers)
	go c.wfInformer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.wfInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	for i := 0; i < workflowTTLWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
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
	key, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(key)

	runtimeutil.HandleError(c.deleteWorkflow(key.(string)))

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
	remaining, ok := c.expiresIn(wf)
	if !ok {
		return
	}
	// if we try and delete in the next second, it is almost certain that the informer is out of sync. Because we
	// double-check that sees if the workflow in the informer is already deleted and we'll make 2 API requests when
	// one is enough.
	// Additionally, this allows enough time to make sure the double checking that the workflow is actually expired
	// truly works.
	addAfter := remaining + time.Second
	key, _ := cache.MetaNamespaceKeyFunc(obj)
	log.Infof("Queueing %v workflow %s for delete in %v", wf.Status.Phase, key, addAfter.Truncate(time.Second))
	c.workqueue.AddAfter(key, addAfter)
}

func (c *Controller) deleteWorkflow(key string) error {
	// It should be impossible for a workflow to have been queue without a valid key.
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)
	// Any workflow that was queued must need deleting, therefore we do not check the expiry again.
	log.Infof("Deleting TTL expired workflow '%s'", key)
	err := c.wfclientset.ArgoprojV1alpha1().Workflows(namespace).Delete(name, &metav1.DeleteOptions{PropagationPolicy: commonutil.GetDeletePropagation()})
	if err != nil {
		if apierr.IsNotFound(err) {
			log.Infof("Workflow already deleted '%s'", key)
		} else {
			return err
		}
	} else {
		log.Infof("Successfully deleted '%s'", key)
	}
	return nil
}

// if the workflow both has a TTL and is expired
func (c *Controller) ttlExpired(wf *wfv1.Workflow) bool {
	expiresIn, ok := c.expiresIn(wf)
	if !ok {
		return false
	}
	return expiresIn <= 0
}

// expiresIn - seconds from now the workflow expires in, maybe <= 0
// ok - if the workflow has a TTL
func (c *Controller) expiresIn(wf *wfv1.Workflow) (expiresIn time.Duration, ok bool) {
	ttl, ok := ttl(wf)
	if !ok {
		return 0, false
	}
	expiresAt := wf.Status.FinishedAt.Add(ttl)
	return expiresAt.Sub(c.clock.Now()), true
}

// ttl - the workflow's TTL
// ok - if the workflow has a TTL
func ttl(wf *wfv1.Workflow) (ttl time.Duration, ok bool) {
	ttlStrategy := wf.GetTTLStrategy()
	if ttlStrategy != nil {
		if wf.Status.Failed() && ttlStrategy.SecondsAfterFailure != nil {
			return time.Duration(*ttlStrategy.SecondsAfterFailure) * time.Second, true
		} else if wf.Status.Successful() && ttlStrategy.SecondsAfterSuccess != nil {
			return time.Duration(*ttlStrategy.SecondsAfterSuccess) * time.Second, true
		} else if wf.Status.Phase.Completed() && ttlStrategy.SecondsAfterCompletion != nil {
			return time.Duration(*ttlStrategy.SecondsAfterCompletion) * time.Second, true
		}
	}
	return 0, false
}
