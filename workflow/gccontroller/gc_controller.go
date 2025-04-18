package gccontroller

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/clock"

	"github.com/argoproj/argo-workflows/v3/config"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	commonutil "github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

var ticker *time.Ticker = time.NewTicker(50 * time.Millisecond)

type Controller struct {
	wfclientset      wfclientset.Interface
	wfInformer       cache.SharedIndexInformer
	workqueue        workqueue.DelayingInterface
	clock            clock.WithTickerAndDelayedExecution
	metrics          *metrics.Metrics
	orderedQueueLock sync.Mutex
	orderedQueue     map[wfv1.WorkflowPhase]*gcHeap
	retentionPolicy  *config.RetentionPolicy
}

// NewController returns a new workflow ttl controller
func NewController(ctx context.Context, wfClientset wfclientset.Interface, wfInformer cache.SharedIndexInformer, metrics *metrics.Metrics, retentionPolicy *config.RetentionPolicy) *Controller {
	orderedQueue := map[wfv1.WorkflowPhase]*gcHeap{
		wfv1.WorkflowFailed:    NewHeap(),
		wfv1.WorkflowError:     NewHeap(),
		wfv1.WorkflowSucceeded: NewHeap(),
	}
	controller := &Controller{
		wfclientset:     wfClientset,
		wfInformer:      wfInformer,
		workqueue:       metrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultControllerRateLimiter(), "workflow_ttl_queue"),
		clock:           clock.RealClock{},
		metrics:         metrics,
		orderedQueue:    orderedQueue,
		retentionPolicy: retentionPolicy,
	}

	_, err := wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			return ok && common.IsDone(un)
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: controller.enqueueWF,
			UpdateFunc: func(old, new interface{}) {
				controller.enqueueWF(new)
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			return ok && common.IsDone(un)
		},
		Handler: cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(old, new interface{}) {
				controller.retentionEnqueue(new)
			},
			AddFunc: func(obj interface{}) {
				controller.retentionEnqueue(obj)
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	return controller
}

func (c *Controller) retentionEnqueue(obj interface{}) {
	// No need to queue the workflow if the retention policy is not set
	if c.retentionPolicy == nil {
		return
	}

	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("'%v' is not an unstructured", obj)
		return
	}

	switch phase := wfv1.WorkflowPhase(un.GetLabels()[common.LabelKeyPhase]); phase {
	case wfv1.WorkflowSucceeded, wfv1.WorkflowFailed, wfv1.WorkflowError:
		c.orderedQueueLock.Lock()
		heap.Push(c.orderedQueue[phase], un)
		c.runGC(phase)
		c.orderedQueueLock.Unlock()
	}
}

func (c *Controller) Run(stopCh <-chan struct{}, workflowGCWorkers int) error {
	defer runtimeutil.HandleCrash()
	defer c.workqueue.ShutDown()
	defer ticker.Stop()
	log.Infof("Starting workflow garbage collector controller (retentionWorkers %d)", workflowGCWorkers)
	go c.wfInformer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.wfInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < workflowGCWorkers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	log.Info("Started workflow garbage collection")
	<-stopCh
	log.Info("Shutting workflow garbage collection")
	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	ctx := context.Background()
	for c.processNextWorkItem(ctx) {
	}
}

// retentionGC queues workflows for deletion based upon the retention policy.
func (c *Controller) runGC(phase wfv1.WorkflowPhase) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	var maxWorkflows int
	switch phase {
	case wfv1.WorkflowSucceeded:
		maxWorkflows = c.retentionPolicy.Completed
	case wfv1.WorkflowFailed:
		maxWorkflows = c.retentionPolicy.Failed
	case wfv1.WorkflowError:
		maxWorkflows = c.retentionPolicy.Errored
	default:
		return
	}

	for c.orderedQueue[phase].Len() > maxWorkflows {
		key, _ := cache.MetaNamespaceKeyFunc(heap.Pop(c.orderedQueue[phase]))
		log.Infof("Queueing %v workflow %s for delete due to max rention(%d workflows)", phase, key, maxWorkflows)
		c.workqueue.Add(key)
		<-ticker.C
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem(ctx context.Context) bool {
	key, quit := c.workqueue.Get()
	if quit {
		return false
	}
	defer c.workqueue.Done(key)
	runtimeutil.HandleError(c.deleteWorkflow(ctx, key.(string)))

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
	log.Infof("Queueing %v workflow %s for delete in %v due to TTL", wf.Status.Phase, key, addAfter.Truncate(time.Second))
	c.workqueue.AddAfter(key, addAfter)
}

func (c *Controller) deleteWorkflow(ctx context.Context, key string) error {
	// It should be impossible for a workflow to have been queue without a valid key.
	namespace, name, _ := cache.SplitMetaNamespaceKey(key)

	// Double check that this workflow is still completed. If it were retried, it may be running again (c.f. https://github.com/argoproj/argo-workflows/issues/12636)
	obj, exists, err := c.wfInformer.GetStore().GetByKey(key)
	if err != nil {
		return nil
	}
	if exists {
		un, ok := obj.(*unstructured.Unstructured)
		if ok && !common.IsDone(un) {
			log.Infof("Workflow '%s' is not completed due to a retry operation, ignore deletion", key)
			return nil
		}
	}

	// Any workflow that was queued must need deleting, therefore we do not check the expiry again.
	log.Infof("Deleting garbage collected workflow '%s'", key)
	err = c.wfclientset.ArgoprojV1alpha1().Workflows(namespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: commonutil.GetDeletePropagation()})
	if err != nil {
		if apierr.IsNotFound(err) {
			log.Infof("Workflow already deleted '%s'", key)
		} else {
			return err
		}
	} else {
		log.Infof("Successfully request '%s' to be deleted", key)
	}
	return nil
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
