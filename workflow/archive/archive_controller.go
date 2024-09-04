package archive

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	syncpkg "github.com/argoproj/pkg/sync"
	log "github.com/sirupsen/logrus"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/persist/sqldb"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/hydrator"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
	"github.com/argoproj/argo-workflows/v3/workflow/util"
)

type Controller struct {
	wfclientset     wfclientset.Interface
	wfInformer      cache.SharedIndexInformer
	wfArchiveQueue  workqueue.RateLimitingInterface
	hydrator        hydrator.Interface
	wfArchive       sqldb.WorkflowArchive
	workflowKeyLock *syncpkg.KeyLock
	metrics         *metrics.Metrics
}

// NewController returns a new workflow archive controller
func NewController(ctx context.Context, wfClientset wfclientset.Interface, wfInformer cache.SharedIndexInformer, metrics *metrics.Metrics, hydrator hydrator.Interface, wfArchive sqldb.WorkflowArchive, workflowKeyLock *syncpkg.KeyLock) *Controller {
	controller := &Controller{
		wfclientset:     wfClientset,
		wfInformer:      wfInformer,
		wfArchiveQueue:  metrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultControllerRateLimiter(), "workflow_archive_queue"),
		metrics:         metrics,
		hydrator:        hydrator,
		wfArchive:       wfArchive,
		workflowKeyLock: workflowKeyLock,
	}

	_, err := wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			// no need to check the `common.LabelKeyCompleted` as we already know it must be complete
			return ok && un.GetLabels()[common.LabelKeyWorkflowArchivingStatus] == "Pending"
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					controller.wfArchiveQueue.Add(key)
				}
			},
			UpdateFunc: func(_, obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					controller.wfArchiveQueue.Add(key)
				}
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	return controller
}

func (c *Controller) Run(stopCh <-chan struct{}, wfArchiveWorkers int) error {
	defer runtimeutil.HandleCrash()
	defer c.wfArchiveQueue.ShutDown()
	log.Infof("Starting workflow archive controller (workflowArchiveWorkers %d)", wfArchiveWorkers)
	go c.wfInformer.Run(stopCh)
	if ok := cache.WaitForCacheSync(stopCh, c.wfInformer.HasSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < wfArchiveWorkers; i++ {
		go wait.Until(c.runArchiveWorker, time.Second, stopCh)
	}
	log.Info("Started workflow archive controller")
	<-stopCh
	log.Info("Shutting workflow archive controller")
	return nil
}

func (c *Controller) runArchiveWorker() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	ctx := context.Background()
	for c.processNextArchiveItem(ctx) {
	}
}

func (c *Controller) processNextArchiveItem(ctx context.Context) bool {
	key, quit := c.wfArchiveQueue.Get()
	if quit {
		return false
	}
	defer c.wfArchiveQueue.Done(key)

	obj, exists, err := c.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get workflow from informer")
		return true
	}
	if !exists {
		return true
	}

	c.archiveWorkflow(ctx, obj)
	return true
}

func (c *Controller) archiveWorkflow(ctx context.Context, obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed to get key for object")
		return
	}
	(*c.workflowKeyLock).Lock(key)
	defer (*c.workflowKeyLock).Unlock(key)
	key, err = cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed to get key for object after locking")
		return
	}
	err = c.archiveWorkflowAux(ctx, obj)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("failed to archive workflow")
	}
}

func (c *Controller) archiveWorkflowAux(ctx context.Context, obj interface{}) error {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return fmt.Errorf("failed to convert to workflow from unstructured: %w", err)
	}
	err = c.hydrator.Hydrate(wf)
	if err != nil {
		return fmt.Errorf("failed to hydrate workflow: %w", err)
	}
	log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name, "uid": wf.UID}).Info("archiving workflow")
	err = c.wfArchive.ArchiveWorkflow(wf)
	if err != nil {
		return fmt.Errorf("failed to archive workflow: %w", err)
	}
	data, err := json.Marshal(map[string]interface{}{
		"metadata": metav1.ObjectMeta{
			Labels: map[string]string{
				common.LabelKeyWorkflowArchivingStatus: "Archived",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal patch: %w", err)
	}
	_, err = c.wfclientset.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(
		ctx,
		un.GetName(),
		types.MergePatchType,
		data,
		metav1.PatchOptions{},
	)
	if err != nil {
		// from this point on we have successfully archived the workflow, and it is possible for the workflow to have actually
		// been deleted, so it's not a problem to get a `IsNotFound` error
		if apierr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to archive workflow: %w", err)
	}
	return nil
}
