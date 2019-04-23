package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/workflow/common"
	"github.com/argoproj/argo/workflow/metrics"
	"github.com/argoproj/argo/workflow/ttlcontroller"
	"github.com/argoproj/argo/workflow/util"
)

// WorkflowController is the controller for workflow resources
type WorkflowController struct {
	// namespace of the workflow controller
	namespace string
	// configMap is the name of the config map in which to derive configuration of the controller from
	configMap string
	// Config is the workflow controller's configuration
	Config WorkflowControllerConfig

	// cliExecutorImage is the executor image as specified from the command line
	cliExecutorImage string

	// cliExecutorImagePullPolicy is the executor imagePullPolicy as specified from the command line
	cliExecutorImagePullPolicy string

	// restConfig is used by controller to send a SIGUSR1 to the wait sidecar using remotecommand.NewSPDYExecutor().
	restConfig    *rest.Config
	kubeclientset kubernetes.Interface
	wfclientset   wfclientset.Interface

	// datastructures to support the processing of workflows and workflow pods
	wfInformer    cache.SharedIndexInformer
	podInformer   cache.SharedIndexInformer
	wfQueue       workqueue.RateLimitingInterface
	podQueue      workqueue.RateLimitingInterface
	completedPods chan string
	throttler     Throttler
}

const (
	workflowResyncPeriod        = 20 * time.Minute
	workflowMetricsResyncPeriod = 1 * time.Minute
	podResyncPeriod             = 30 * time.Minute
)

// NewWorkflowController instantiates a new WorkflowController
func NewWorkflowController(
	restConfig *rest.Config,
	kubeclientset kubernetes.Interface,
	wfclientset wfclientset.Interface,
	namespace,
	executorImage,
	executorImagePullPolicy,
	configMap string,
) *WorkflowController {
	wfc := WorkflowController{
		restConfig:                 restConfig,
		kubeclientset:              kubeclientset,
		wfclientset:                wfclientset,
		configMap:                  configMap,
		namespace:                  namespace,
		cliExecutorImage:           executorImage,
		cliExecutorImagePullPolicy: executorImagePullPolicy,
		wfQueue:                    workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		podQueue:                   workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		completedPods:              make(chan string, 512),
	}
	wfc.throttler = NewThrottler(0, wfc.wfQueue)
	return &wfc
}

// MetricsServer starts a prometheus metrics server if enabled in the configmap
func (wfc *WorkflowController) MetricsServer(ctx context.Context) {
	if wfc.Config.MetricsConfig.Enabled {
		informer := util.NewWorkflowInformer(wfc.restConfig, wfc.Config.Namespace, workflowMetricsResyncPeriod, wfc.tweakWorkflowMetricslist)
		go informer.Run(ctx.Done())
		registry := metrics.NewWorkflowRegistry(informer)
		metrics.RunServer(ctx, wfc.Config.MetricsConfig, registry)
	}
}

// TelemetryServer starts a prometheus telemetry server if enabled in the configmap
func (wfc *WorkflowController) TelemetryServer(ctx context.Context) {
	if wfc.Config.TelemetryConfig.Enabled {
		registry := metrics.NewTelemetryRegistry()
		metrics.RunServer(ctx, wfc.Config.TelemetryConfig, registry)
	}
}

// RunTTLController runs the workflow TTL controller
func (wfc *WorkflowController) RunTTLController(ctx context.Context) {
	ttlCtrl := ttlcontroller.NewController(
		wfc.restConfig,
		wfc.wfclientset,
		wfc.Config.Namespace,
		wfc.Config.InstanceID,
	)
	err := ttlCtrl.Run(ctx.Done())
	if err != nil {
		panic(err)
	}
}

// Run starts an Workflow resource controller
func (wfc *WorkflowController) Run(ctx context.Context, wfWorkers, podWorkers int) {
	defer wfc.wfQueue.ShutDown()
	defer wfc.podQueue.ShutDown()

	log.Infof("Workflow Controller (version: %s) starting", argo.GetVersion())
	log.Infof("Workers: workflow: %d, pod: %d", wfWorkers, podWorkers)
	log.Info("Watch Workflow controller config map updates")
	_, err := wfc.watchControllerConfigMap(ctx)
	if err != nil {
		log.Errorf("Failed to register watch for controller config map: %v", err)
		return
	}

	wfc.wfInformer = util.NewWorkflowInformer(wfc.restConfig, wfc.Config.Namespace, workflowResyncPeriod, wfc.tweakWorkflowlist)

	wfc.addWorkflowInformerHandler()
	wfc.podInformer = wfc.newPodInformer()

	go wfc.wfInformer.Run(ctx.Done())
	go wfc.podInformer.Run(ctx.Done())
	go wfc.podLabeler(ctx.Done())

	// Wait for all involved caches to be synced, before processing items from the queue is started
	for _, informer := range []cache.SharedIndexInformer{wfc.wfInformer, wfc.podInformer} {
		if !cache.WaitForCacheSync(ctx.Done(), informer.HasSynced) {
			log.Error("Timed out waiting for caches to sync")
			return
		}
	}

	for i := 0; i < wfWorkers; i++ {
		go wait.Until(wfc.runWorker, time.Second, ctx.Done())
	}
	for i := 0; i < podWorkers; i++ {
		go wait.Until(wfc.podWorker, time.Second, ctx.Done())
	}
	<-ctx.Done()
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

	if wf.ObjectMeta.Labels[common.LabelKeyCompleted] == "true" {
		wfc.throttler.Remove(key)
		// can get here if we already added the completed=true label,
		// but we are still draining the controller's workflow workqueue
		return true
	}

	woc := newWorkflowOperationCtx(wf, wfc)

	// Decompress the node if it is compressed
	err = util.DecompressWorkflow(woc.wf)
	if err != nil {
		woc.log.Warnf("workflow decompression failed: %v", err)
		woc.markWorkflowFailed(fmt.Sprintf("workflow decompression failed: %s", err.Error()))
		woc.persistUpdates()
		wfc.throttler.Remove(key)
		return true
	}
	woc.operate()
	if woc.wf.Status.Completed() {
		wfc.throttler.Remove(key)
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
	// TODO: currently we reawaken the workflow on *any* pod updates.
	// But this could be be much improved to become smarter by only
	// requeue the workflow when there are changes that we care about.
	wfc.wfQueue.Add(pod.ObjectMeta.Namespace + "/" + workflowName)
	return true
}

func (wfc *WorkflowController) tweakWorkflowlist(options *metav1.ListOptions) {
	options.FieldSelector = fields.Everything().String()
	// completed notin (true)
	incompleteReq, err := labels.NewRequirement(common.LabelKeyCompleted, selection.NotIn, []string{"true"})
	if err != nil {
		panic(err)
	}
	labelSelector := labels.NewSelector().
		Add(*incompleteReq).
		Add(util.InstanceIDRequirement(wfc.Config.InstanceID))
	options.LabelSelector = labelSelector.String()
}

func (wfc *WorkflowController) tweakWorkflowMetricslist(options *metav1.ListOptions) {
	options.FieldSelector = fields.Everything().String()
	labelSelector := labels.NewSelector().Add(util.InstanceIDRequirement(wfc.Config.InstanceID))
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

func (wfc *WorkflowController) addWorkflowInformerHandler() {
	wfc.wfInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					wfc.wfQueue.Add(key)
					priority, creation := getWfPriority(obj)
					wfc.throttler.Add(key, priority, creation)
				}
			},
			UpdateFunc: func(old, new interface{}) {
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
	)
}

func (wfc *WorkflowController) newWorkflowPodWatch() *cache.ListWatch {
	c := wfc.kubeclientset.CoreV1().RESTClient()
	resource := "pods"
	namespace := wfc.Config.Namespace
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
