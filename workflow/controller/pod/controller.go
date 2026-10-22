// Package pod reconciles pods and takes care of gc events
package pod

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/diff"
	envutil "github.com/argoproj/argo-workflows/v4/util/env"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

const (
	podResyncPeriod      = 30 * time.Minute
	podPaginationLimit   = 500
	heldPodRecheckPeriod = time.Minute
)

var (
	incompleteReq, _ = labels.NewRequirement(common.LabelKeyCompleted, selection.Equals, []string{"false"})
	workflowReq, _   = labels.NewRequirement(common.LabelKeyWorkflow, selection.Exists, nil)
	keyFunc          = cache.DeletionHandlingMetaNamespaceKeyFunc
)

type podEventCallback func(pod *apiv1.Pod) error

// Controller is a controller for pods
type Controller struct {
	config        *argoConfig.Config
	kubeclientset kubernetes.Interface
	wfInformer    cache.SharedIndexInformer
	workqueue     workqueue.TypedRateLimitingInterface[string]
	podInformer   cache.SharedIndexInformer
	callBack      podEventCallback
	log           logging.Logger
	restConfig    *rest.Config
	// how often recheckHeldPod looks at a deleting pod held for its workflow
	recheckPeriod time.Duration
	// how long a deleting pod that finished is held for its workflow at most, counted from its deletion or
	// from startedAt, whichever is later
	heldPodTimeout time.Duration
	startedAt      time.Time
}

// NewController creates a pod controller
func NewController(ctx context.Context, config *argoConfig.Config, restConfig *rest.Config, namespace string, clientSet kubernetes.Interface, wfInformer cache.SharedIndexInformer, metrics *metrics.Metrics, callback podEventCallback) *Controller {
	ctx, log := logging.RequireLoggerFromContext(ctx).WithField("component", "pod_controller").InContext(ctx)
	podController := &Controller{
		config:        config,
		kubeclientset: clientSet,
		wfInformer:    wfInformer,
		workqueue:     metrics.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultTypedControllerRateLimiter[string](), "pod_cleanup_queue"),
		podInformer:   newInformer(clientSet, &config.InstanceID, &namespace),
		log:           log,
		callBack:      callback,
		restConfig:    restConfig,
		recheckPeriod: heldPodRecheckPeriod,
		// the time the workflow waits for a task result, see taskResultReconciliation
		heldPodTimeout: envutil.LookupEnvDurationOr(ctx, "TASK_RESULT_TIMEOUT_DURATION", 10*time.Minute),
		startedAt:      time.Now(),
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

// Check if owned pod's workflow no longer exists or workflow is in deletion
func (c *Controller) podOrphaned(ctx context.Context, pod *apiv1.Pod) bool {
	controllerRef := metav1.GetControllerOf(pod)
	// Pod had no owner
	if controllerRef == nil ||
		controllerRef.Kind != workflow.WorkflowKind {
		return false
	}
	wfOwnerKey := fmt.Sprintf("%s/%s", pod.Namespace, controllerRef.Name)
	ctx, log := c.log.WithFields(logging.Fields{"wfOwnerKey": wfOwnerKey, "namespace": pod.Namespace, "pod": pod.Name}).InContext(ctx)
	obj, wfExists, err := c.wfInformer.GetIndexer().GetByKey(wfOwnerKey)
	if err != nil {
		log.Warn(ctx, "failed to get workflow from informer")
	}
	if !wfExists {
		return true
	}
	un, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Warn(ctx, "workflow is not an unstructured")
		return true
	}
	wf, err := util.FromUnstructured(un)
	if err != nil {
		log.Warn(ctx, "workflow unstructured can't be converted to a workflow")
		return true
	}
	return wf.DeletionTimestamp != nil
}

// ownerWorkflowActive reports whether the pod's owning workflow (same UID) is in the informer, not being
// deleted and not completed, i.e. it will still reconcile this pod. Reads metadata only.
func (c *Controller) ownerWorkflowActive(pod *apiv1.Pod) bool {
	controllerRef := metav1.GetControllerOf(pod)
	if controllerRef == nil || controllerRef.Kind != workflow.WorkflowKind {
		return false
	}
	obj, exists, err := c.wfInformer.GetIndexer().GetByKey(pod.Namespace + "/" + controllerRef.Name)
	if err != nil || !exists {
		return false
	}
	wf, ok := obj.(metav1.Object)
	return ok && wf.GetUID() == controllerRef.UID && wf.GetDeletionTimestamp() == nil &&
		wf.GetLabels()[common.LabelKeyCompleted] != "true"
}

// heldPodReleasable reports whether a deleting pod held for its workflow may lose its status finalizer: its
// workflow is no longer active; or no kubelet finished the pod 2m after its deletionTimestamp, so none is left
// to report a result; or the workflow still has not taken the result of the finished pod after heldPodTimeout
// (e.g. its task result never completes).
func (c *Controller) heldPodReleasable(pod *apiv1.Pod) bool {
	if !c.ownerWorkflowActive(pod) {
		return true
	}
	if !podFinished(pod) {
		return time.Now().After(pod.DeletionTimestamp.Add(2 * time.Minute))
	}
	since := pod.DeletionTimestamp.Time
	if c.startedAt.After(since) {
		since = c.startedAt
	}
	return time.Now().After(since.Add(c.heldPodTimeout))
}

// podFinished reports whether the kubelet finished the pod. PodGC marks the pod of a lost node Failed while its
// containers are still reported running.
func podFinished(pod *apiv1.Pod) bool {
	switch pod.Status.Phase {
	case apiv1.PodSucceeded:
		return true
	case apiv1.PodFailed:
		for _, c := range pod.Status.ContainerStatuses {
			if (c.Name == common.WaitContainerName || c.Name == common.SupervisorContainerName) && c.State.Terminated == nil {
				return false
			}
		}
		return true
	}
	return false
}

func podGCFromPod(pod *apiv1.Pod) wfv1.PodGC {
	if val, ok := pod.Annotations[common.AnnotationKeyPodGCStrategy]; ok {
		strategy, delay, _ := strings.Cut(val, "/")
		return wfv1.PodGC{Strategy: wfv1.PodGCStrategy(strategy), DeleteDelayDuration: delay}
	}
	return wfv1.PodGC{Strategy: wfv1.PodGCOnPodNone}
}

// Returns time.IsZero if no last transition
func podLastTransition(pod *apiv1.Pod) time.Time {
	lastTransition := time.Time{}
	for _, condition := range pod.Status.Conditions {
		if condition.LastTransitionTime.After(lastTransition) {
			lastTransition = condition.LastTransitionTime.Time
		}
	}
	return lastTransition
}

// A common handler for
func (c *Controller) commonPodEvent(ctx context.Context, pod *apiv1.Pod, deleting bool) {
	// All pods here are not marked completed
	action := noAction
	minimumDelay := time.Duration(0)
	podGC := podGCFromPod(pod)
	switch {
	case deleting:
		if hasOurFinalizer(pod.Finalizers) {
			if os.Getenv(common.EnvVarPodStatusCaptureFinalizer) == "true" && pod.Status.Phase != apiv1.PodPending && c.ownerWorkflowActive(pod) {
				// The workflow removes the finalizer once it has persisted the pod's node (persistUpdates ->
				// queuePodsForCleanup), so a timer here could race that write. A Pending pod has no result to
				// wait for. recheckHeldPod releases the pod if its workflow is deleted, completed or replaced, if
				// no kubelet finished it 2m after its deletionTimestamp (e.g. force deleted on an unreachable
				// node), or if the workflow has not taken its result after heldPodTimeout.
				c.log.WithFields(logging.Fields{"namespace": pod.Namespace, "podName": pod.Name}).Info(ctx, "Leaving finalizer for the workflow to remove")
				c.queuePodForCleanupAfter(ctx, pod.Namespace, pod.Name, recheckHeldPod, c.recheckPeriod)
				return
			}
			c.log.WithFields(logging.Fields{"pod.Finalizers": pod.Finalizers}).Info(ctx, "Removing finalizers during a delete")
			action = removeFinalizer
			minimumDelay = 2 * time.Minute
		}
	case c.podOrphaned(ctx, pod):
		if hasOurFinalizer(pod.Finalizers) {
			action = removeFinalizer
		}
		switch {
		case podGC.Strategy == wfv1.PodGCOnWorkflowCompletion:
		case podGC.Strategy == wfv1.PodGCOnPodCompletion:
		case podGC.Strategy == wfv1.PodGCOnPodSuccess && pod.Status.Phase == apiv1.PodSucceeded:
			action = deletePod
		}
	}
	if action != noAction {
		// The workflow is gone, we have no idea when that happened, so lets base around pod transiution
		lastTransition := podLastTransition(pod)
		// GetDeleteDelayDuration returns -1 if no duration, we don't care about failure to parse otherwise here
		delay := time.Duration(0)
		delayDuration, _ := podGC.GetDeleteDelayDuration()
		// In the case of a raw delete make sure we've had some time to process it if there was a finalizer
		if delayDuration < minimumDelay {
			delayDuration = minimumDelay
		}
		if !lastTransition.IsZero() && delayDuration > 0 {
			delay = time.Until(lastTransition.Add(delayDuration))
		}
		c.log.WithFields(logging.Fields{"action": action, "namespace": pod.Namespace, "podName": pod.Name, "podGC": podGC, "delay": delay}).Info(ctx, "queuing pod delay")
		switch {
		case delay > 0:
			c.queuePodForCleanupAfter(ctx, pod.Namespace, pod.Name, action, delay)
		default:
			c.queuePodForCleanup(ctx, pod.Namespace, pod.Name, action)
		}
	}
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
