// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	argoConfig "github.com/argoproj/argo-workflows/v4/config"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/diff"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

const (
	podResyncPeriod    = 30 * time.Minute
	podPaginationLimit = 500
)

var (
	incompleteReq, _ = labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	workflowReq, _   = labels.NewRequirement(common.LabelKeyWorkflow, selection.Exists, nil)
	keyFunc          = cache.DeletionHandlingMetaNamespaceKeyFunc
)

type podEventCallback func(pod *apiv1.Pod) error
type workflowLookupCallback func(ctx context.Context, namespace, name string, hydrateNodes bool) (*wfv1.Workflow, error)

// Controller is a controller for pods
type Controller struct {
	config             *argoConfig.Config
	kubeclientset      kubernetes.Interface
	wfInformer         cache.SharedIndexInformer
	workqueue          workqueue.TypedRateLimitingInterface[string]
	podInformer        cache.SharedIndexInformer
	callBack           podEventCallback
	lookupWorkflow     workflowLookupCallback
	hydrateWorkflow    func(context.Context, *wfv1.Workflow) error
	signalContainer    containerSignalFunc
	recaptureLegacy    LegacyPodRecapture
	cleanupDiagnostics cleanupDiagnostics
	log                logging.Logger
	restConfig         *rest.Config
}

// NewController creates a pod controller
func NewController(ctx context.Context, config *argoConfig.Config, restConfig *rest.Config, namespace string, clientSet kubernetes.Interface, wfInformer cache.SharedIndexInformer, metrics *metrics.Metrics, callback podEventCallback, lookupWorkflow workflowLookupCallback) *Controller {
	ctx, log := logging.RequireLoggerFromContext(ctx).WithField("component", "pod_controller").InContext(ctx)
	podController := &Controller{
		config:         config,
		kubeclientset:  clientSet,
		wfInformer:     wfInformer,
		workqueue:      metrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultTypedControllerRateLimiter[string](), "pod_cleanup_queue"),
		podInformer:    newInformer(clientSet, &config.InstanceID, &namespace),
		log:            log,
		callBack:       callback,
		lookupWorkflow: lookupWorkflow,
		restConfig:     restConfig,
	}
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	podController.podInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj any) {
				pod, err := podFromObj(obj)
				if err != nil {
					log.WithError(err).Error(ctx, "object from informer wasn't a pod")
					return
				}
				podController.addPodEvent(ctx, pod)
			},
			UpdateFunc: func(old, newVal any) {
				key, err := keyFunc(newVal)
				if err != nil {
					return
				}
				oldPod, newPod := old.(*apiv1.Pod), newVal.(*apiv1.Pod)
				if oldPod.ResourceVersion == newPod.ResourceVersion {
					return
				}
				if !significantPodChange(oldPod, newPod) {
					log.WithField("key", key).Debug(ctx, "insignificant pod change")
					diff.LogChanges(ctx, oldPod, newPod)
					return
				}
				podController.updatePodEvent(ctx, oldPod, newPod)
			},
			DeleteFunc: func(obj any) {
				podController.deletePodEvent(ctx, obj)
			},
		},
	)
	return podController
}

func (c *Controller) HasSynced() func() bool {
	return c.podInformer.HasSynced
}

// Run runs the pod controller
func (c *Controller) Run(ctx context.Context, workers int) {
	defer c.workqueue.ShutDown()
	if !cache.WaitForCacheSync(ctx.Done(), c.wfInformer.HasSynced) {
		return
	}
	go c.podInformer.Run(ctx.Done())
	if !cache.WaitForCacheSync(ctx.Done(), c.HasSynced()) {
		return
	}
	for range workers {
		go wait.UntilWithContext(ctx, c.runPodCleanup, time.Second)
	}
	<-ctx.Done()
}

// GetPodPhaseMetrics obtains pod metrics
func (c *Controller) GetPodPhaseMetrics(ctx context.Context) map[string]int64 {
	result := make(map[string]int64, 0)
	if c.podInformer != nil {
		for _, phase := range []apiv1.PodPhase{apiv1.PodRunning, apiv1.PodPending} {
			objs, err := c.podInformer.GetIndexer().IndexKeys(indexes.PodPhaseIndex, string(phase))
			if err != nil {
				c.log.WithField("phase", phase).WithError(err).Error(ctx, "failed to list pods in phase")
			} else {
				result[string(phase)] = int64(len(objs))
			}
		}
	}
	return result
}

func podGCFromPod(pod *apiv1.Pod) wfv1.PodGC {
	if val, ok := pod.Annotations[common.AnnotationKeyPodGCStrategy]; ok {
		strategy, delay, _ := strings.Cut(val, "/")
		return wfv1.PodGC{Strategy: wfv1.PodGCStrategy(strategy), DeleteDelayDuration: delay}
	}
	return wfv1.PodGC{Strategy: wfv1.PodGCOnPodNone}
}

// Pod events are recovery hints; only the worker can authorize cleanup from
// authoritative owner and node state. Startup Add events restore lost work.
func (c *Controller) commonPodEvent(ctx context.Context, pod *apiv1.Pod, _ bool) {
	c.ReconcilePodCleanup(ctx, pod)
}

func (c *Controller) addPodEvent(ctx context.Context, pod *apiv1.Pod) {
	err := c.callBack(pod)
	if err != nil {
		c.log.WithField("pod", pod.Name).Warn(ctx, "callback for pod add failed")
	}
	deleting := pod.DeletionTimestamp != nil
	c.commonPodEvent(ctx, pod, deleting)
}

func (c *Controller) updatePodEvent(ctx context.Context, _ *apiv1.Pod, newPod *apiv1.Pod) {
	// This is only called for actual updates, where there are "significant changes"
	err := c.callBack(newPod)
	if err != nil {
		c.log.WithField("pod", newPod.Name).Warn(ctx, "callback for pod update failed")
	}
	deleting := newPod.DeletionTimestamp != nil
	c.commonPodEvent(ctx, newPod, deleting)
}

func (c *Controller) deletePodEvent(ctx context.Context, obj any) {
	pod, err := podFromObj(obj)
	if err != nil {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			c.log.Info(ctx, "error obtaining pod object from tombstone")
			return
		}
		pod, ok = tombstone.Obj.(*apiv1.Pod)
		if !ok {
			c.log.Warn(ctx, "deleted pod last known state not a pod")
			return
		}
	}
	// enqueue the workflow for the deleted pod
	err = c.callBack(pod)
	if err != nil {
		c.log.WithField("pod", pod.Name).Warn(ctx, "callback for pod delete failed")
	}
	// Backstop to remove finalizer if it hasn't already happened, our last chance
	c.commonPodEvent(ctx, pod, true)
}

func newWorkflowPodWatch(clientSet kubernetes.Interface, instanceID, namespace *string) *cache.ListWatch {
	c := clientSet.CoreV1().Pods(*namespace)
	// completed=false
	labelSelector := labels.NewSelector().
		Add(*workflowReq).
		// not sure if we should do this
		Add(*incompleteReq).
		Add(util.InstanceIDRequirement(*instanceID))

	listFunc := func(ctx context.Context, options metav1.ListOptions) (runtime.Object, error) {
		options.LabelSelector = labelSelector.String()
		var allPods []apiv1.Pod
		continueTok := ""
		options.Limit = podPaginationLimit
		for {
			options.Continue = continueTok
			if options.Continue != "" {
				options.ResourceVersion = ""
				options.ResourceVersionMatch = ""
			}
			podList, err := c.List(ctx, options)
			if err != nil {
				return nil, err
			}
			allPods = append(allPods, podList.Items...)
			if podList.Continue == "" {
				break
			}
			continueTok = podList.Continue
		}
		return &apiv1.PodList{Items: allPods}, nil
	}
	watchFunc := func(ctx context.Context, options metav1.ListOptions) (watch.Interface, error) {
		options.Watch = true
		options.LabelSelector = labelSelector.String()
		return c.Watch(ctx, options)
	}
	return &cache.ListWatch{ListWithContextFunc: listFunc, WatchFuncWithContext: watchFunc}
}

func newInformer(clientSet kubernetes.Interface, instanceID, namespace *string) cache.SharedIndexInformer {
	source := newWorkflowPodWatch(clientSet, instanceID, namespace)
	informer := cache.NewSharedIndexInformer(cache.ToListWatcherWithWatchListSemantics(source, clientSet), &apiv1.Pod{}, podResyncPeriod, cache.Indexers{
		indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
		indexes.NodeIDIndex:   indexes.MetaNodeIDIndexFunc,
		indexes.PodPhaseIndex: indexes.PodPhaseIndexFunc,
	})
	//nolint:errcheck // the error only happens if the informer was already started, and it hasn't been
	informer.SetTransform(informerutil.StripManagedFields)
	return informer
}

func podFromObj(obj any) (*apiv1.Pod, error) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		return nil, fmt.Errorf("object is not a pod")
	}
	return pod, nil
}
