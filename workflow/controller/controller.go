package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/pkg/errors"
	syncpkg "github.com/argoproj/pkg/sync"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	v1Label "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	apiwatch "k8s.io/client-go/tools/watch"
	"k8s.io/client-go/util/workqueue"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo/v2"
	"github.com/argoproj/argo/v2/config"
	argoErr "github.com/argoproj/argo/v2/errors"
	"github.com/argoproj/argo/v2/persist/sqldb"
	wfv1 "github.com/argoproj/argo/v2/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/v2/pkg/client/clientset/versioned"
	wfextvv1alpha1 "github.com/argoproj/argo/v2/pkg/client/informers/externalversions/workflow/v1alpha1"
	authutil "github.com/argoproj/argo/v2/util/auth"
	errorsutil "github.com/argoproj/argo/v2/util/errors"
	"github.com/argoproj/argo/v2/workflow/common"
	controllercache "github.com/argoproj/argo/v2/workflow/controller/cache"
	"github.com/argoproj/argo/v2/workflow/controller/estimation"
	"github.com/argoproj/argo/v2/workflow/controller/indexes"
	"github.com/argoproj/argo/v2/workflow/controller/informer"
	"github.com/argoproj/argo/v2/workflow/controller/pod"
	"github.com/argoproj/argo/v2/workflow/cron"
	"github.com/argoproj/argo/v2/workflow/events"
	"github.com/argoproj/argo/v2/workflow/hydrator"
	"github.com/argoproj/argo/v2/workflow/metrics"
	"github.com/argoproj/argo/v2/workflow/sync"
	"github.com/argoproj/argo/v2/workflow/ttlcontroller"
	"github.com/argoproj/argo/v2/workflow/util"
)

// WorkflowController is the controller for workflow resources
type WorkflowController struct {
	// namespace of the workflow controller
	namespace        string
	managedNamespace string

	configController config.Controller
	// Config is the workflow controller's configuration
	Config config.Config

	// cliExecutorImage is the executor image as specified from the command line
	cliExecutorImage string

	// cliExecutorImagePullPolicy is the executor imagePullPolicy as specified from the command line
	cliExecutorImagePullPolicy string
	containerRuntimeExecutor   string

	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig       *rest.Config
	kubeclientset    kubernetes.Interface
	dynamicInterface dynamic.Interface
	wfclientset      wfclientset.Interface

	// datastructures to support the processing of workflows and workflow pods
	wfInformer            cache.SharedIndexInformer
	wftmplInformer        wfextvv1alpha1.WorkflowTemplateInformer
	cwftmplInformer       wfextvv1alpha1.ClusterWorkflowTemplateInformer
	podInformer           cache.SharedIndexInformer
	wfQueue               workqueue.RateLimitingInterface
	podQueue              workqueue.RateLimitingInterface
	completedPods         chan string
	gcPods                chan string // pods to be deleted depend on GC strategy
	throttler             sync.Throttler
	workflowKeyLock       syncpkg.KeyLock // used to lock workflows for exclusive modification or access
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	hydrator              hydrator.Interface
	wfArchive             sqldb.WorkflowArchive
	estimatorFactory      estimation.EstimatorFactory
	syncManager           *sync.Manager
	metrics               *metrics.Metrics
	eventRecorderManager  events.EventRecorderManager
	archiveLabelSelector  labels.Selector
	cacheFactory          controllercache.Factory
}

const (
	workflowResyncPeriod                = 20 * time.Minute
	workflowTemplateResyncPeriod        = 20 * time.Minute
	podResyncPeriod                     = 30 * time.Minute
	clusterWorkflowTemplateResyncPeriod = 20 * time.Minute
)

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(restConfig *rest.Config, kubeclientset kubernetes.Interface, wfclientset wfclientset.Interface, namespace, managedNamespace, executorImage, executorImagePullPolicy, containerRuntimeExecutor, configMap string) (*WorkflowController, error) {
	dynamicInterface, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	wfc := WorkflowController{
		restConfig:                 restConfig,
		kubeclientset:              kubeclientset,
		dynamicInterface:           dynamicInterface,
		wfclientset:                wfclientset,
		namespace:                  namespace,
		managedNamespace:           managedNamespace,
		cliExecutorImage:           executorImage,
		cliExecutorImagePullPolicy: executorImagePullPolicy,
		containerRuntimeExecutor:   containerRuntimeExecutor,
		configController:           config.NewController(namespace, configMap, kubeclientset, config.EmptyConfigFunc),
		completedPods:              make(chan string, 512),
		gcPods:                     make(chan string, 512),
		workflowKeyLock:            syncpkg.NewKeyLock(),
		cacheFactory:               controllercache.NewCacheFactory(kubeclientset, namespace),
		eventRecorderManager:       events.NewEventRecorderManager(kubeclientset),
	}

	wfc.UpdateConfig()

	wfc.metrics = metrics.New(wfc.getMetricsServerConfig())

	workqueue.SetProvider(wfc.metrics) // must execute SetProvider before we created the queues
	wfc.wfQueue = workqueue.NewNamedRateLimitingQueue(&fixedItemIntervalRateLimiter{}, "workflow_queue")
	wfc.throttler = wfc.newThrottler()
	wfc.podQueue = workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "pod_queue")

	return &wfc, nil
}

func (wfc *WorkflowController) newThrottler() sync.Throttler {
	return sync.NewThrottler(wfc.Config.Parallelism, func(key string) { wfc.wfQueue.AddRateLimited(key) })
}

// RunTTLController runs the workflow TTL controller
func (wfc *WorkflowController) runTTLController(ctx context.Context, workflowTTLWorkers int) {
	ttlCtrl := ttlcontroller.NewController(wfc.wfclientset, wfc.wfInformer)
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	err := ttlCtrl.Run(ctx.Done(), workflowTTLWorkers)
	if err != nil {
		panic(err)
	}
}

func (wfc *WorkflowController) runCronController(ctx context.Context) {
	cronController := cron.NewCronController(wfc.wfclientset, wfc.dynamicInterface, wfc.namespace, wfc.GetManagedNamespace(), wfc.Config.InstanceID, wfc.metrics, wfc.eventRecorderManager)
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	cronController.Run(ctx)
}

var indexers = cache.Indexers{
	indexes.ClusterWorkflowTemplateIndex: indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyClusterWorkflowTemplate),
	indexes.CronWorkflowIndex:            indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyCronWorkflow),
	indexes.WorkflowTemplateIndex:        indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyWorkflowTemplate),
	indexes.SemaphoreConfigIndexName:     indexes.WorkflowSemaphoreKeysIndexFunc(),
	indexes.WorkflowPhaseIndex:           indexes.MetaWorkflowPhaseIndexFunc(),
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context, wfWorkers, workflowTTLWorkers, podWorkers int) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()

	log.WithField("version", argo.GetVersion().Version).Info("Starting Workflow Controller")
	log.Infof("Workers: workflow: %d, pod: %d", wfWorkers, podWorkers)

	wfc.wfInformer = util.NewWorkflowInformer(wfc.dynamicInterface, wfc.GetManagedNamespace(), workflowResyncPeriod, wfc.tweakListOptions, indexers)
	wfc.wftmplInformer = informer.NewTolerantWorkflowTemplateInformer(wfc.dynamicInterface, workflowTemplateResyncPeriod, wfc.managedNamespace)

	wfc.addWorkflowInformerHandlers()
	wfc.podInformer = wfc.newPodInformer()
	wfc.updateEstimatorFactory()

	go wfc.runConfigMapWatcher(ctx.Done())
	go wfc.configController.Run(ctx.Done(), wfc.updateConfig)
	go wfc.wfInformer.Run(ctx.Done())
	go wfc.wftmplInformer.Informer().Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())
	go wfc.podLabeler(ctx.Done())
	go wfc.podGarbageCollector(ctx.Done())
	go wfc.workflowGarbageCollector(ctx.Done())
	go wfc.archivedWorkflowGarbageCollector(ctx.Done())

	go wfc.runTTLController(ctx, workflowTTLWorkers)
	go wfc.runCronController(ctx)

	go wfc.metrics.RunServer(ctx)
	go wait.Until(wfc.syncWorkflowPhaseMetrics, 15*time.Second, ctx.Done())

	wfc.createClusterWorkflowTemplateInformer(ctx)
	wfc.waitForCacheSync(ctx)

	// Create Synchronization Manager
	err := wfc.createSynchronizationManager()
	if err != nil {
		panic(err)
	}

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
}

func (wfc *WorkflowController) waitForCacheSync(ctx context.Context) {
	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(ctx.Done(), wfc.wfInformer.HasSynced, wfc.wftmplInformer.Informer().HasSynced, wfc.podInformer.HasSynced) {
		panic("Timed out waiting for caches to sync")
	}
	if wfc.cwftmplInformer != nil {
		if !cache.WaitForCacheSync(ctx.Done(), wfc.cwftmplInformer.Informer().HasSynced) {
			panic("Timed out waiting for caches to sync")
		}
	}
}

// Create and initialize the Synchronization Manager
func (wfc *WorkflowController) createSynchronizationManager() error {
	getSyncLimit := func(lockKey string) (int, error) {
		lockName, err := sync.DecodeLockName(lockKey)
		if err != nil {
			return 0, err
		}
		configMap, err := wfc.kubeclientset.CoreV1().ConfigMaps(lockName.Namespace).Get(lockName.ResourceName, metav1.GetOptions{})
		if err != nil {
			return 0, err
		}

		value, found := configMap.Data[lockName.Key]
		if !found {
			return 0, argoErr.New(argoErr.CodeBadRequest, fmt.Sprintf("Sync configuration key '%s' not found in ConfigMap", lockName.Key))
		}
		return strconv.Atoi(value)
	}

	nextWorkflow := func(key string) {
		wfc.wfQueue.AddRateLimited(key)
	}

	wfc.syncManager = sync.NewLockManager(getSyncLimit, nextWorkflow)

	labelSelector := v1Label.NewSelector()
	req, _ := v1Label.NewRequirement(common.LabelKeyPhase, selection.Equals, []string{string(wfv1.NodeRunning)})
	if req != nil {
		labelSelector = labelSelector.Add(*req)
	}

	listOpts := metav1.ListOptions{LabelSelector: labelSelector.String()}
	wfList, err := wfc.wfclientset.ArgoprojV1alpha1().Workflows(wfc.namespace).List(listOpts)
	if err != nil {
		return err
	}

	wfc.syncManager.Initialize(wfList.Items)
	return nil
}

func (wfc *WorkflowController) runConfigMapWatcher(stopCh <-chan struct{}) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

	retryWatcher, err := apiwatch.NewRetryWatcher("1", &cache.ListWatch{
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return wfc.kubeclientset.CoreV1().ConfigMaps(wfc.managedNamespace).Watch(metav1.ListOptions{})
		},
	})
	if err != nil {
		panic(err)
	}
	defer retryWatcher.Stop()

	for {
		select {
		case event := <-retryWatcher.ResultChan():
			cm, ok := event.Object.(*apiv1.ConfigMap)
			if !ok {
				log.Errorf("invalid config map object received in config watcher. Ignored processing")
				continue
			}
			log.Debugf("received config map %s/%s update", cm.Namespace, cm.Name)
			wfc.notifySemaphoreConfigUpdate(cm)

		case <-stopCh:
			return
		}
	}
}

// notifySemaphoreConfigUpdate will notify semaphore config update to pending workflows
func (wfc *WorkflowController) notifySemaphoreConfigUpdate(cm *apiv1.ConfigMap) {
	wfs, err := wfc.wfInformer.GetIndexer().ByIndex(indexes.SemaphoreConfigIndexName, fmt.Sprintf("%s/%s", cm.Namespace, cm.Name))
	if err != nil {
		log.Errorf("failed get the workflow from informer. %v", err)
	}

	for _, obj := range wfs {
		un, ok := obj.(*unstructured.Unstructured)
		if !ok {
			log.Warnf("received object from indexer %s is not an unstructured", indexes.SemaphoreConfigIndexName)
			continue
		}
		wfc.wfQueue.AddRateLimited(fmt.Sprintf("%s/%s", un.GetNamespace(), un.GetName()))
	}
}

// Check if the controller has RBAC access to ClusterWorkflowTemplates
func (wfc *WorkflowController) createClusterWorkflowTemplateInformer(ctx context.Context) {
	cwftGetAllowed, err := authutil.CanI(wfc.kubeclientset, "get", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)
	cwftListAllowed, err := authutil.CanI(wfc.kubeclientset, "list", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)
	cwftWatchAllowed, err := authutil.CanI(wfc.kubeclientset, "watch", "clusterworkflowtemplates", wfc.namespace, "")
	errors.CheckError(err)

	if cwftGetAllowed && cwftListAllowed && cwftWatchAllowed {
		wfc.cwftmplInformer = informer.NewTolerantClusterWorkflowTemplateInformer(wfc.dynamicInterface, clusterWorkflowTemplateResyncPeriod)
		go wfc.cwftmplInformer.Informer().Run(ctx.Done())
	} else {
		log.Warnf("Controller doesn't have RBAC access for ClusterWorkflowTemplates")
	}
}

func (wfc *WorkflowController) UpdateConfig() {
	config, err := wfc.configController.Get()
	if err != nil {
		log.Fatalf("Failed to register watch for controller config map: %v", err)
	}
	err = wfc.updateConfig(config)
	if err != nil {
		log.Fatalf("Failed to update config: %v", err)
	}
}

// podLabeler will label all pods on the controllers completedPod channel as completed
func (wfc *WorkflowController) podLabeler(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case pod := <-wfc.completedPods:
			parts := strings.Split(pod, "/")
			if len(parts) != 2 {
				log.WithFields(log.Fields{"pod": pod}).Warn("Unexpected item on completed pod channel")
				continue
			}
			namespace := parts[0]
			podName := parts[1]
			err := common.AddPodLabel(wfc.kubeclientset, podName, namespace, common.LabelKeyCompleted, "true")
			if err != nil {
				if !apierr.IsNotFound(err) {
					log.WithFields(log.Fields{"namespace": namespace, "pod": podName, "err": err}).Error("Failed to labeled pod completed")
				}
			} else {
				log.WithFields(log.Fields{"namespace": namespace, "pod": podName}).Info("Labeled pod completed")
			}
		}
	}
}

// podGarbageCollector will delete all pods on the controllers gcPods channel as completed
func (wfc *WorkflowController) podGarbageCollector(stopCh <-chan struct{}) {
	for {
		select {
		case <-stopCh:
			return
		case pod := <-wfc.gcPods:
			parts := strings.Split(pod, "/")
			if len(parts) != 2 {
				log.WithFields(log.Fields{"pod": pod}).Warn("Unexpected item on gcPods channel")
				continue
			}
			namespace := parts[0]
			podName := parts[1]
			err := common.DeletePod(wfc.kubeclientset, podName, namespace)
			if err != nil {
				log.WithFields(log.Fields{"namespace": namespace, "pod": podName, "err": err}).Error("Failed to delete pod for gc")
			} else {
				log.WithFields(log.Fields{"namespace": namespace, "pod": podName}).Info("Delete pod for gc successfully")
			}
		}
	}
}

func (wfc *WorkflowController) workflowGarbageCollector(stopCh <-chan struct{}) {
	value, ok := os.LookupEnv("WORKFLOW_GC_PERIOD")
	periodicity := 5 * time.Minute
	if ok {
		var err error
		periodicity, err = time.ParseDuration(value)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "value": value}).Fatal("Failed to parse WORKFLOW_GC_PERIOD")
		}
	}
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	log.Infof("Performing periodic GC every %v", periodicity)
	ticker := time.NewTicker(periodicity)
	for {
		select {
		case <-stopCh:
			ticker.Stop()
			return
		case <-ticker.C:
			if wfc.offloadNodeStatusRepo.IsEnabled() {
				log.Info("Performing periodic workflow GC")
				oldRecords, err := wfc.offloadNodeStatusRepo.ListOldOffloads(wfc.GetManagedNamespace())
				if err != nil {
					log.WithField("err", err).Error("Failed to list old offloaded nodes")
					continue
				}
				if len(oldRecords) == 0 {
					log.Info("Zero old offloads, nothing to do")
					continue
				}
				// get every lives workflow (1000s) into a map
				liveOffloadNodeStatusVersions := make(map[types.UID]string)
				workflows, err := util.NewWorkflowLister(wfc.wfInformer).List()
				if err != nil {
					log.WithField("err", err).Error("Failed to list incomplete workflows")
					continue
				}
				for _, wf := range workflows {
					// this could be the empty string - as it is no longer offloaded
					liveOffloadNodeStatusVersions[wf.UID] = wf.Status.OffloadNodeStatusVersion
				}
				log.WithFields(log.Fields{"len_wfs": len(liveOffloadNodeStatusVersions), "len_old_offloads": len(oldRecords)}).Info("Deleting old offloads that are not live")
				for _, record := range oldRecords {
					// this could be empty string
					nodeStatusVersion, ok := liveOffloadNodeStatusVersions[types.UID(record.UID)]
					if !ok || nodeStatusVersion != record.Version {
						err := wfc.offloadNodeStatusRepo.Delete(record.UID, record.Version)
						if err != nil {
							log.WithField("err", err).Error("Failed to delete offloaded nodes")
						}
					}
				}
			}
		}
	}
}

func (wfc *WorkflowController) archivedWorkflowGarbageCollector(stopCh <-chan struct{}) {
	value, ok := os.LookupEnv("ARCHIVED_WORKFLOW_GC_PERIOD")
	periodicity := 24 * time.Hour
	if ok {
		var err error
		periodicity, err = time.ParseDuration(value)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "value": value}).Fatal("Failed to parse ARCHIVED_WORKFLOW_GC_PERIOD")
		}
	}
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	if wfc.Config.Persistence == nil {
		log.Info("Persistence disabled - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	if !wfc.Config.Persistence.Archive {
		log.Info("Archive disabled - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	ttl := wfc.Config.Persistence.ArchiveTTL
	if ttl == config.TTL(0) {
		log.Info("Archived workflows TTL zero - so archived workflow GC disabled - you must restart the controller if you enable this")
		return
	}
	log.WithFields(log.Fields{"ttl": ttl, "periodicity": periodicity}).Info("Performing archived workflow GC")
	ticker := time.NewTicker(periodicity)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			log.Info("Performing archived workflow GC")
			err := wfc.wfArchive.DeleteExpiredWorkflows(time.Duration(ttl))
			if err != nil {
				log.WithField("err", err).Error("Failed to delete archived workflows")
			}
		}
	}
}

func (wfc *WorkflowController) runWorker() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	for wfc.processNextItem() {
	}
}

// processNextItem is the worker logic for handling workflow updates
func (wfc *WorkflowController) processNextItem() bool {
	key, quit := wfc.wfQueue.Get()
	if quit {
		return false
	}
	defer wfc.wfQueue.Done(key)

	obj, exists, err := wfc.wfInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get workflow from informer")
		return true
	}
	if !exists {
		// This happens after a workflow was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}

	wfc.workflowKeyLock.Lock(key.(string))
	defer wfc.workflowKeyLock.Unlock(key.(string))

	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.WithFields(log.Fields{"key": key}).Warn("Index is not an unstructured")
		return true
	}

	if !wfc.throttler.Admit(key.(string)) {
		log.WithFields(log.Fields{"key": key}).Info("Workflow processing has been postponed due to max parallelism limit")
		return true
	}

	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Warn("Failed to unmarshal key to workflow object")
		woc := newWorkflowOperationCtx(wf, wfc)
		woc.markWorkflowFailed(fmt.Sprintf("cannot unmarshall spec: %s", err.Error()))
		woc.persistUpdates()
		return true
	}

	if wf.Labels[common.LabelKeyCompleted] == "true" {
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return true
	}
	// this will ensure we process every incomplete workflow once every 20m
	wfc.wfQueue.AddAfter(key, workflowResyncPeriod)

	woc := newWorkflowOperationCtx(wf, wfc)

	// make sure this is removed from the throttler is complete
	defer func() {
		// must be done with woc
		if woc.wf.Labels[common.LabelKeyCompleted] == "true" {
			wfc.throttler.Remove(key.(string))
		}
	}()

	err = wfc.hydrator.Hydrate(woc.wf)
	if err != nil {
		transientErr := errorsutil.IsTransientErr(err)
		woc.log.WithField("transientErr", transientErr).Errorf("hydration failed: %v", err)
		if !transientErr {
			woc.markWorkflowError(err)
			woc.persistUpdates()
		}
		return true
	}

	startTime := time.Now()
	woc.operate()
	wfc.metrics.OperationCompleted(time.Since(startTime).Seconds())
	if woc.wf.Status.Fulfilled() {
		// Send all completed pods to gcPods channel to delete it later depend on the PodGCStrategy.
		var doPodGC bool
		if woc.execWf.Spec.PodGC != nil {
			switch woc.execWf.Spec.PodGC.Strategy {
			case wfv1.PodGCOnWorkflowCompletion:
				doPodGC = true
			case wfv1.PodGCOnWorkflowSuccess:
				if woc.wf.Status.Successful() {
					doPodGC = true
				}
			}
		}
		if doPodGC {
			for podName := range woc.completedPods {
				pod := fmt.Sprintf("%s/%s", woc.wf.ObjectMeta.Namespace, podName)
				woc.controller.gcPods <- pod
			}
		}
	}

	// TODO: operate should return error if it was unable to operate properly
	// so we can requeue the work for a later time
	// See: https://github.com/kubernetes/client-go/blob/master/examples/workqueue/main.go
	//c.handleErr(err, key)
	return true
}

func (wfc *WorkflowController) podWorker() {
	for wfc.processNextPodItem() {
	}
}

// processNextPodItem is the worker logic for handling pod updates.
// For pods updates, this simply means to "wake up" the workflow by
// adding the corresponding workflow key into the workflow workqueue.
func (wfc *WorkflowController) processNextPodItem() bool {
	key, quit := wfc.podQueue.Get()
	if quit {
		return false
	}
	defer wfc.podQueue.Done(key)

	obj, exists, err := wfc.podInformer.GetIndexer().GetByKey(key.(string))
	if err != nil {
		log.WithFields(log.Fields{"key": key, "error": err}).Error("Failed to get pod from informer index")
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}

	err = wfc.enqueueWfFromPodLabel(obj)
	if err != nil {
		log.WithError(err).Warnf("Failed to enqueue the workflow for %s", key)
	}
	return true
}

// enqueueWfFromPodLabel will extract the workflow name from pod label and
// enqueue workflow for processing
func (wfc *WorkflowController) enqueueWfFromPodLabel(obj interface{}) error {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		return fmt.Errorf("Key in index is not a pod")
	}
	if pod.Labels == nil {
		return fmt.Errorf("Pod did not have labels")
	}
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		return fmt.Errorf("Watch returned pod unrelated to any workflow")
	}
	wfc.wfQueue.AddRateLimited(pod.ObjectMeta.Namespace + "/" + workflowName)
	return nil
}

func (wfc *WorkflowController) tweakListOptions(options *metav1.ListOptions) {
	labelSelector := labels.NewSelector().
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))
	options.LabelSelector = labelSelector.String()
}

func getWfPriority(obj interface{}) (int32, time.Time) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return 0, time.Now()
	}
	priority, hasPriority, err := unstructured.NestedInt64(un.Object, "spec", "priority")
	if err != nil {
		return 0, un.GetCreationTimestamp().Time
	}
	if !hasPriority {
		priority = 0
	}

	return int32(priority), un.GetCreationTimestamp().Time
}

func (wfc *WorkflowController) addWorkflowInformerHandlers() {
	wfc.wfInformer.AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				return !common.UnstructuredHasCompletedLabel(obj)
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc: func(obj interface{}) {
					key, err := cache.MetaNamespaceKeyFunc(obj)
					if err == nil {
						// for a new workflow, we do not want to rate limit its execution using AddRateLimited
						wfc.wfQueue.AddAfter(key, wfc.Config.InitialDelay.Duration)
						priority, creation := getWfPriority(obj)
						wfc.throttler.Add(key, priority, creation)
					}
				},
				UpdateFunc: func(old, new interface{}) {
					oldWf, newWf := old.(*unstructured.Unstructured), new.(*unstructured.Unstructured)
					// this check is very important to prevent doing many reconciliations we do not need to do
					if oldWf.GetResourceVersion() == newWf.GetResourceVersion() {
						return
					}
					key, err := cache.MetaNamespaceKeyFunc(new)
					if err == nil {
						wfc.wfQueue.AddRateLimited(key)
						priority, creation := getWfPriority(new)
						wfc.throttler.Add(key, priority, creation)
					}
				},
				DeleteFunc: func(obj interface{}) {
					// IndexerInformer uses a delta queue, therefore for deletes we have to use this
					// key function.
					key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
					if err == nil {
						wfc.releaseAllWorkflowLocks(obj)
						// no need to add to the queue - this workflow is done
						wfc.throttler.Remove(key)
					}
				},
			},
		},
	)
	wfc.wfInformer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			un, ok := obj.(*unstructured.Unstructured)
			// no need to check the `common.LabelKeyCompleted` as we already know it must be complete
			return ok && un.GetLabels()[common.LabelKeyWorkflowArchivingStatus] == "Pending"
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: wfc.archiveWorkflow,
			UpdateFunc: func(_, obj interface{}) {
				wfc.archiveWorkflow(obj)
			},
		},
	},
	)
	wfc.wfInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			wf, ok := obj.(*unstructured.Unstructured)
			if ok { // maybe cache.DeletedFinalStateUnknown
				wfc.metrics.StopRealtimeMetricsForKey(string(wf.GetUID()))
			}
		},
	})
}

func (wfc *WorkflowController) archiveWorkflow(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		log.Error("failed to get key for object")
		return
	}
	wfc.workflowKeyLock.Lock(key)
	defer wfc.workflowKeyLock.Unlock(key)
	err = wfc.archiveWorkflowAux(obj)
	if err != nil {
		log.WithField("key", key).WithError(err).Error("failed to archive workflow")
	}
}

func (wfc *WorkflowController) archiveWorkflowAux(obj interface{}) error {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		return fmt.Errorf("failed to convert to workflow from unstructured: %w", err)
	}
	err = wfc.hydrator.Hydrate(wf)
	if err != nil {
		return fmt.Errorf("failed to hydrate workflow: %w", err)
	}
	log.WithFields(log.Fields{"namespace": wf.Namespace, "workflow": wf.Name, "uid": wf.UID}).Info("archiving workflow")
	err = wfc.wfArchive.ArchiveWorkflow(wf)
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
	_, err = wfc.wfclientset.ArgoprojV1alpha1().Workflows(un.GetNamespace()).Patch(un.GetName(), types.MergePatchType, data)
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

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().Pods(wfc.GetManagedNamespace())
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		return c.List(options)
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = labelSelector.String()
		return c.Watch(options)
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newPodInformer() cache.SharedIndexInformer {
	source := wfc.newWorkflowPodWatch()
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{
		indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
	})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err != nil {
					return
				}
				wfc.podQueue.Add(key)
			},
			UpdateFunc: func(old, new interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err != nil {
					return
				}
				oldPod, newPod := old.(*apiv1.Pod), new.(*apiv1.Pod)
				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				if !pod.SignificantPodChange(oldPod, newPod) {
					log.WithField("key", key).Info("insignificant pod change")
					pod.LogChanges(oldPod, newPod)
					return
				}
				wfc.podQueue.Add(key)
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.

				// Enqueue the workflow for deleted pod
				_ = wfc.enqueueWfFromPodLabel(obj)

			},
		},
	)
	return informer
}

// call this func whenever the configuration changes, or when the workflow informer changes
func (wfc *WorkflowController) updateEstimatorFactory() {
	wfc.estimatorFactory = estimation.NewEstimatorFactory(wfc.wfInformer, wfc.hydrator, wfc.wfArchive)
}

// setWorkflowDefaults sets values in the workflow.Spec with defaults from the
// workflowController. Values in the workflow will be given the upper hand over the defaults.
// The defaults for the workflow controller are set in the workflow-controller config map
func (wfc *WorkflowController) setWorkflowDefaults(wf *wfv1.Workflow) error {
	if wfc.Config.WorkflowDefaults != nil {
		err := util.MergeTo(wfc.Config.WorkflowDefaults, wf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) GetManagedNamespace() string {
	if wfc.managedNamespace != "" {
		return wfc.managedNamespace
	}
	return wfc.Config.Namespace
}

func (wfc *WorkflowController) GetContainerRuntimeExecutor() string {
	if wfc.containerRuntimeExecutor != "" {
		return wfc.containerRuntimeExecutor
	}
	return wfc.Config.ContainerRuntimeExecutor
}

func (wfc *WorkflowController) getMetricsServerConfig() (metrics.ServerConfig, metrics.ServerConfig) {
	// Metrics config
	path := wfc.Config.MetricsConfig.Path
	if path == "" {
		path = metrics.DefaultMetricsServerPath
	}
	port := wfc.Config.MetricsConfig.Port
	if port == 0 {
		port = metrics.DefaultMetricsServerPort
	}
	metricsConfig := metrics.ServerConfig{
		Enabled:      wfc.Config.MetricsConfig.Enabled == nil || *wfc.Config.MetricsConfig.Enabled,
		Path:         path,
		Port:         port,
		TTL:          time.Duration(wfc.Config.MetricsConfig.MetricsTTL),
		IgnoreErrors: wfc.Config.MetricsConfig.IgnoreErrors,
	}

	// Telemetry config
	path = metricsConfig.Path
	if wfc.Config.TelemetryConfig.Path != "" {
		path = wfc.Config.TelemetryConfig.Path
	}

	port = metricsConfig.Port
	if wfc.Config.TelemetryConfig.Port > 0 {
		port = wfc.Config.TelemetryConfig.Port
	}
	telemetryConfig := metrics.ServerConfig{
		Enabled:      wfc.Config.TelemetryConfig.Enabled == nil || *wfc.Config.TelemetryConfig.Enabled,
		Path:         path,
		Port:         port,
		IgnoreErrors: wfc.Config.TelemetryConfig.IgnoreErrors,
	}
	return metricsConfig, telemetryConfig
}

func (wfc *WorkflowController) releaseAllWorkflowLocks(obj interface{}) {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.WithFields(log.Fields{"key": obj}).Warn("Key in index is not an unstructured")
		return
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.WithFields(log.Fields{"key": obj}).Warn("Invalid workflow object")
		return
	}
	if wf.Status.Synchronization != nil {
		wfc.syncManager.ReleaseAll(wf)
	}
}

func (wfc *WorkflowController) isArchivable(wf *wfv1.Workflow) bool {
	return wfc.archiveLabelSelector.Matches(labels.Set(wf.Labels))
}

func (wfc *WorkflowController) syncWorkflowPhaseMetrics() {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)
	for _, phase := range []wfv1.NodePhase{wfv1.NodePending, wfv1.NodeRunning, wfv1.NodeSucceeded, wfv1.NodeFailed, wfv1.NodeError} {
		objs, err := wfc.wfInformer.GetIndexer().ByIndex(indexes.WorkflowPhaseIndex, string(phase))
		if err != nil {
			log.WithError(err).Errorf("failed to list workflows by '%s'", phase)
			continue
		}
		wfc.metrics.SetWorkflowPhaseGauge(phase, len(objs))
	}
}
