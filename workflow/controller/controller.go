package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/argoproj/pkg/errors"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"upper.io/db.v3/lib/sqlbuilder"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/persist/sqldb"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	wfextv "github.com/argoproj/argo/pkg/client/informers/externalversions"
	wfextvv1alpha1 "github.com/argoproj/argo/pkg/client/informers/externalversions/workflow/v1alpha1"
	authutil "github.com/argoproj/argo/util/auth"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/controller/pod"
	"github.com/argoproj/argo/workflow/cron"
	"github.com/argoproj/argo/workflow/hydrator"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/ttlcontroller"
	"github.com/argoproj/argo/workflow/util"
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
	restConfig    *rest.Config
	kubeclientset kubernetes.Interface
	wfclientset   wfclientset.Interface

	// datastructures to support the processing of workflows and workflow pods
	wfInformer            cache.SharedIndexInformer
	wftmplInformer        wfextvv1alpha1.WorkflowTemplateInformer
	cwftmplInformer       wfextvv1alpha1.ClusterWorkflowTemplateInformer
	podInformer           cache.SharedIndexInformer
	wfQueue               workqueue.RateLimitingInterface
	podQueue              workqueue.RateLimitingInterface
	completedPods         chan string
	gcPods                chan string // pods to be deleted depend on GC strategy
	throttler             Throttler
	session               sqlbuilder.Database
	offloadNodeStatusRepo sqldb.OffloadNodeStatusRepo
	hydrator              hydrator.Interface
	wfArchive             sqldb.WorkflowArchive
	metrics               metrics.Metrics
	eventRecorderManager  EventRecorderManager
	archiveLabelSelector  labels.Selector
}

const (
	workflowResyncPeriod                = 20 * time.Minute
	workflowTemplateResyncPeriod        = 20 * time.Minute
	podResyncPeriod                     = 30 * time.Minute
	clusterWorkflowTemplateResyncPeriod = 20 * time.Minute
)

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(restConfig *rest.Config, kubeclientset kubernetes.Interface, wfclientset wfclientset.Interface, namespace string, managedNamespace string, executorImage, executorImagePullPolicy, containerRuntimeExecutor, configMap string) *WorkflowController {
	wfc := WorkflowController{
		restConfig:                 restConfig,
		kubeclientset:              kubeclientset,
		wfclientset:                wfclientset,
		namespace:                  namespace,
		managedNamespace:           managedNamespace,
		cliExecutorImage:           executorImage,
		cliExecutorImagePullPolicy: executorImagePullPolicy,
		containerRuntimeExecutor:   containerRuntimeExecutor,
		wfQueue:                    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		configController:           config.NewController(namespace, configMap, kubeclientset),
		completedPods:              make(chan string, 512),
		gcPods:                     make(chan string, 512),
		eventRecorderManager:       newEventRecorderManager(kubeclientset),
	}
	wfc.throttler = NewThrottler(0, wfc.wfQueue)
	wfc.UpdateConfig()

	wfc.metrics = metrics.New(wfc.getMetricsServerConfig())
	return &wfc
}

// RunTTLController runs the workflow TTL controller
func (wfc *WorkflowController) runTTLController(ctx context.Context) {
	ttlCtrl := ttlcontroller.NewController(wfc.wfclientset, wfc.wfInformer)
	err := ttlCtrl.Run(ctx.Done())
	if err != nil {
		panic(err)
	}
}

func (wfc *WorkflowController) runCronController(ctx context.Context) {
	cronController := cron.NewCronController(wfc.wfclientset, wfc.restConfig, wfc.namespace, wfc.GetManagedNamespace(), wfc.Config.InstanceID, &wfc.metrics)
	cronController.Run(ctx)
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context, wfWorkers, podWorkers int) {
	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()

	log.WithField("version", argo.GetVersion().Version).Info("Starting Workflow Controller")
	log.Infof("Workers: workflow: %d, pod: %d", wfWorkers, podWorkers)

	wfc.wfInformer = util.NewWorkflowInformer(wfc.restConfig, wfc.GetManagedNamespace(), workflowResyncPeriod, wfc.tweakListOptions)
	wfc.wftmplInformer = wfc.newWorkflowTemplateInformer()

	wfc.addWorkflowInformerHandlers()
	wfc.podInformer = wfc.newPodInformer()

	go wfc.configController.Run(ctx.Done(), wfc.updateConfig)
	go wfc.wfInformer.Run(ctx.Done())
	go wfc.wftmplInformer.Informer().Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())
	go wfc.podLabeler(ctx.Done())
	go wfc.podGarbageCollector(ctx.Done())
	go wfc.workflowGarbageCollector(ctx.Done())
	go wfc.archivedWorkflowGarbageCollector(ctx.Done())

	go wfc.runTTLController(ctx)
	go wfc.runCronController(ctx)
	go wfc.metrics.RunServer(ctx)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, informer := range []cache.SharedIndexInformer{wfc.wfInformer, wfc.wftmplInformer.Informer(), wfc.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
	}

	wfc.createClusterWorkflowTemplateInformer(ctx)

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
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
		wfc.cwftmplInformer = wfc.newClusterWorkflowTemplateInformer()
		go wfc.cwftmplInformer.Informer().Run(ctx.Done())
		if !cache.WaitForCacheSync(ctx.Done(), wfc.cwftmplInformer.Informer().HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
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
				log.Warnf("Unexpected item on completed pod channel: %s", pod)
				continue
			}
			namespace := parts[0]
			podName := parts[1]
			err := common.AddPodLabel(wfc.kubeclientset, podName, namespace, common.LabelKeyCompleted, "true")
			if err != nil {
				if !apierr.IsNotFound(err) {
					log.Errorf("Failed to label pod %s/%s completed: %+v", namespace, podName, err)
				}
			} else {
				log.Infof("Labeled pod %s/%s completed", namespace, podName)
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
				log.Warnf("Unexpected item on gcPods channel: %s", pod)
				continue
			}
			namespace := parts[0]
			podName := parts[1]
			err := common.DeletePod(wfc.kubeclientset, podName, namespace)
			if err != nil {
				log.Errorf("Failed to delete pod %s/%s for gc: %+v", namespace, podName, err)
			} else {
				log.Infof("Delete pod %s/%s for gc successfully", namespace, podName)
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
		log.Errorf("Failed to get workflow '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// This happens after a workflow was labeled with completed=true
		// or was deleted, but the work queue still had an entry for it.
		return true
	}
	// The workflow informer receives unstructured objects to deal with the possibility of invalid
	// workflow manifests that are unable to unmarshal to workflow objects
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warnf("Key '%s' in index is not an unstructured", key)
		return true
	}

	if key, ok = wfc.throttler.Next(key); !ok {
		log.Warnf("Workflow %s processing has been postponed due to max parallelism limit", key)
		return true
	}

	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.Warnf("Failed to unmarshal key '%s' to workflow object: %v", key, err)
		woc := newWorkflowOperationCtx(wf, wfc)
		woc.markWorkflowFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
		woc.persistUpdates()
		wfc.throttler.Remove(key)
		return true
	}

	err = wfc.setWorkflowDefaults(wf)
	if err != nil {
		log.Warnf("Failed to apply default workflow values to '%s': %v", wf.Name, err)
		woc := newWorkflowOperationCtx(wf, wfc)
		woc.markWorkflowFailed(fmt.Sprintf("invalid spec: %s", err.Error()))
		woc.persistUpdates()
		wfc.throttler.Remove(key)
		return true
	}

	if wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true" {
		wfc.throttler.Remove(key)
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return true
	}

	woc := newWorkflowOperationCtx(wf, wfc)

	err = wfc.hydrator.Hydrate(woc.wf)
	if err != nil {
		woc.log.Errorf("hydration failed: %v", err)
		woc.markWorkflowError(err, true)
		woc.persistUpdates()
		wfc.throttler.Remove(key)
		return true
	}

	startTime := time.Now()
	woc.operate()
	wfc.metrics.OperationCompleted(time.Since(startTime).Seconds())
	if woc.wf.Status.Fulfilled() {
		wfc.throttler.Remove(key)
		// Send all completed pods to gcPods channel to delete it later depend on the PodGCStrategy.
		var doPodGC bool
		if woc.wfSpec.PodGC != nil {
			switch woc.wfSpec.PodGC.Strategy {
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
		log.Errorf("Failed to get pod '%s' from informer index: %+v", key, err)
		return true
	}
	if !exists {
		// we can get here if pod was queued into the pod workqueue,
		// but it was either deleted or labeled completed by the time
		// we dequeued it.
		return true
	}
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		log.Warnf("Key '%s' in index is not a pod", key)
		return true
	}
	if pod.Labels == nil {
		log.Warnf("Pod '%s' did not have labels", key)
		return true
	}
	workflowName, ok := pod.Labels[common.LabelKeyWorkflow]
	if !ok {
		// Ignore pods unrelated to workflow (this shouldn't happen unless the watch is setup incorrectly)
		log.Warnf("watch returned pod unrelated to any workflow: %s", pod.ObjectMeta.Name)
		return true
	}
	// add this change after 1s - this reduces the number of workflow reconciliations -
	//with each reconciliation doing more work
	wfc.wfQueue.AddAfter(pod.ObjectMeta.Namespace+"/"+workflowName, 1*time.Second)
	return true
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

func getWfPhase(obj interface{}) wfv1.NodePhase {
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return ""
	}
	phase, hasPhase, err := unstructured.NestedString(un.Object, "status", "phase")
	if err != nil {
		return ""
	}
	if !hasPhase {
		return wfv1.NodePending
	}
	return wfv1.NodePhase(phase)
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
						wfc.wfQueue.Add(key)
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
						wfc.wfQueue.Add(key)
						priority, creation := getWfPriority(new)
						wfc.throttler.Add(key, priority, creation)
					}
				},
				DeleteFunc: func(obj interface{}) {
					// IndexerInformer uses a delta queue, therefore for deletes we have to use this
					// key function.
					key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
					if err == nil {
						wfc.wfQueue.Add(key)
						wfc.throttler.Remove(key)
					}
				},
			},
		},
	)
	wfc.wfInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			wfc.metrics.WorkflowAdded(getWfPhase(obj))
		},
		UpdateFunc: func(old, new interface{}) {
			wfc.metrics.WorkflowUpdated(getWfPhase(old), getWfPhase(new))
		},
		DeleteFunc: func(obj interface{}) {
			wfc.metrics.WorkflowDeleted(getWfPhase(obj))
		},
	})
}

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "pods"
	namespace := wfc.GetManagedNamespace()
	// completed=false
	incompleteReq, _ := labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))

	listFunc := func(options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Do().Get()
	}
	watchFunc := func(options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = labelSelector.String()
		req := c.Get().
			Namespace(namespace).
			Resource(resource).
			VersionedParams(&options, metav1.ParameterCodec)
		return req.Watch()
	}
	return &cache.ListWatch{ListFunc: listFunc, WatchFunc: watchFunc}
}

func (wfc *WorkflowController) newPodInformer() cache.SharedIndexInformer {
	source := wfc.newWorkflowPodWatch()
	informer := cache.NewSharedIndexInformer(source, &apiv1.Pod{}, podResyncPeriod, cache.Indexers{})
	informer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			UpdateFunc: func(old, new interface{}) {
				oldPod, newPod := old.(*apiv1.Pod), new.(*apiv1.Pod)
				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				if !pod.SignificantPodChange(oldPod, newPod) {
					return
				}
				key, err := cache.MetaNamespaceKeyFunc(new)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				// IndexerInformer uses a delta queue, therefore for deletes we have to use this
				// key function.
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.podQueue.Add(key)
				}
			},
		},
	)
	return informer
}

// setWorkflowDefaults sets values in the workflow.Spec with defaults from the
// workflowController. Values in the workflow will be given the upper hand over the defaults.
// The defaults for the workflow controller are set in the workflow-controller config map
func (wfc *WorkflowController) setWorkflowDefaults(wf *wfv1.Workflow) error {
	if wfc.Config.WorkflowDefaults != nil {
		defaultsSpec, err := json.Marshal(*wfc.Config.WorkflowDefaults)
		if err != nil {
			return err
		}
		workflowBytes, err := json.Marshal(wf)
		if err != nil {
			return err
		}
		mergedWf, err := strategicpatch.StrategicMergePatch(defaultsSpec, workflowBytes, wfv1.Workflow{})
		if err != nil {
			return err
		}
		err = json.Unmarshal(mergedWf, &wf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (wfc *WorkflowController) newWorkflowTemplateInformer() wfextvv1alpha1.WorkflowTemplateInformer {
	return wfextv.NewSharedInformerFactoryWithOptions(wfc.wfclientset, workflowTemplateResyncPeriod, wfextv.WithNamespace(wfc.GetManagedNamespace())).Argoproj().V1alpha1().WorkflowTemplates()
}

func (wfc *WorkflowController) newClusterWorkflowTemplateInformer() wfextvv1alpha1.ClusterWorkflowTemplateInformer {
	return wfextv.NewSharedInformerFactoryWithOptions(wfc.wfclientset, clusterWorkflowTemplateResyncPeriod).Argoproj().V1alpha1().ClusterWorkflowTemplates()
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
	if port == "" {
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
	if wfc.Config.TelemetryConfig.Port != "" {
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

func (wfc *WorkflowController) isArchivable(wf *wfv1.Workflow) bool {
	return wfc.archiveLabelSelector.Matches(labels.Set(wf.Labels))

}
