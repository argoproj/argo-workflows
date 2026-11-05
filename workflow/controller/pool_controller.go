package controller

import (
	"context"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsinformers "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4"
	authutil "github.com/argoproj/argo-workflows/v4/util/auth"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/poolcontroller"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// poolWorkers is how many pools the pool controller reconciles at once.
const poolWorkers = 2

// Whether hot workers were enabled when the controller started, which is
// what it runs with until it restarts.
const (
	hotWorkersUndecided int32 = iota
	hotWorkersOff
	hotWorkersOn
)

// noDeploymentRBAC is every pool's health when the controller can't manage
// Deployments.
const noDeploymentRBAC = "the workflow controller isn't allowed to manage Deployments, which hot workers need: install the hot workers RBAC component and restart the controller"

// recordHotWorkers reads hotWorkers.enabled once, as the controller starts.
// Dispatch and the pool controller both follow what it read, not the live
// config, so turning the flag on or off takes a controller restart.
func (wfc *WorkflowController) recordHotWorkers() {
	state := hotWorkersOff
	if wfc.Config.HotWorkers.IsEnabled() {
		state = hotWorkersOn
	}
	wfc.hotWorkersAtStart.Store(state)
}

// hotWorkersEnabled is whether hot workers were enabled when the controller
// started.
func (wfc *WorkflowController) hotWorkersEnabled() bool {
	return wfc.hotWorkersAtStart.Load() == hotWorkersOn
}

// setUpPoolController sets up the pool controller and its Deployment
// informer, only if hot workers were enabled when the controller started
// (recordHotWorkers). With the flag off there is no informer, no
// reconciler and no Deployment call, so the controller needs no Deployment
// RBAC. With the flag on but without that RBAC, the pool controller doesn't
// run, and every pool reports degraded with the reason, which pooled nodes
// show while they wait.
func (wfc *WorkflowController) setUpPoolController(ctx context.Context) {
	logger := logging.RequireLoggerFromContext(ctx)
	if !wfc.hotWorkersEnabled() {
		logger.Info(ctx, "Hot workers are disabled: not running the pool controller")
		return
	}
	can, err := authutil.CanI(ctx, wfc.kubeclientset, poolcontroller.DeploymentVerbs, appsv1.GroupName, wfc.GetManagedNamespace(), "deployments")
	if err == nil && !can {
		logger.WithField("verbs", poolcontroller.DeploymentVerbs).Error(ctx, "Hot workers are enabled, but the controller isn't allowed to manage Deployments: not running the pool controller")
		wfc.poolHealth = poolcontroller.Degraded(noDeploymentRBAC)
		return
	}
	if err != nil {
		// The pool controller waits for its informer to sync, so if it can't
		// list Deployments only pools are held up.
		logger.WithError(err).Warn(ctx, "Couldn't check the controller's Deployment RBAC; running the pool controller anyway")
	}
	wfc.poolDeploymentInformer = wfc.newPoolDeploymentInformer(ctx)
	wfc.poolController = poolcontroller.New(poolcontroller.Options{
		KubeClient:        wfc.kubeclientset,
		WfClient:          wfc.wfclientset,
		TaskSets:          wfc.pooledTaskSetInformer,
		Pods:              wfc.poolPodInformer,
		Deployments:       wfc.poolDeploymentInformer,
		Queue:             wfc.metrics.RateLimiterWithBusyWorkers(ctx, poolcontroller.NewRateLimiter(), "pool_queue"),
		InstanceID:        wfc.Config.InstanceID,
		ControllerVersion: argo.GetVersion().Version,
		Build:             buildPoolDeployment,
		DefaultRetention: func() time.Duration {
			return poolcontroller.Retention(wfc.Config.HotWorkers.GetDefaultRetentionSeconds())
		},
		Events:         wfc.eventRecorderManager,
		OnHealthChange: wfc.enqueueWorkflowsWaitingOnPool,
	})
	wfc.poolHealth = wfc.poolController
	logger.Info(ctx, "Hot workers are enabled: running the pool controller")
}

// runPoolController starts the Deployment informer and the pool controller,
// if setUpPoolController set them up.
func (wfc *WorkflowController) runPoolController(ctx context.Context) {
	if wfc.poolController == nil {
		return
	}
	go wfc.poolDeploymentInformer.Run(ctx.Done())
	go wfc.poolController.Run(ctx, poolWorkers)
}

// warnHotWorkersFlagChange warns, on a config update, that a change to
// hotWorkers.enabled takes a controller restart.
func (wfc *WorkflowController) warnHotWorkersFlagChange(ctx context.Context) {
	state := wfc.hotWorkersAtStart.Load()
	if state == hotWorkersUndecided || wfc.Config.HotWorkers.IsEnabled() == (state == hotWorkersOn) {
		return
	}
	logging.RequireLoggerFromContext(ctx).WithField("enabled", wfc.Config.HotWorkers.IsEnabled()).
		Warn(ctx, "hotWorkers.enabled changed: the workflow controller runs with the value it started with until it restarts")
}

// newPoolDeploymentInformer caches this instance's pool Deployments, trimmed
// by transformPoolDeployment, selected as its pooled TaskSets are: by a pool
// hash and its instance ID. It needs the optional Deployment RBAC.
func (wfc *WorkflowController) newPoolDeploymentInformer(ctx context.Context) cache.SharedIndexInformer {
	selector := wfc.poolSelector()
	logging.RequireLoggerFromContext(ctx).WithField("labelSelector", selector).Info(ctx, "Watching pool deployments")
	// resyncPeriod 0: the pool controller requeues an idle pool for the end of
	// its retention itself.
	informer := appsinformers.NewFilteredDeploymentInformer(
		wfc.kubeclientset,
		wfc.GetManagedNamespace(),
		0,
		cache.Indexers{
			indexes.PoolIndex: indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyPoolHash),
		},
		func(options *metav1.ListOptions) {
			options.LabelSelector = selector
		},
	)
	//nolint:errcheck // the error only happens if the informer was already started, and it hasn't been
	informer.SetTransform(transformPoolDeployment)
	return informer
}

// transformPoolDeployment keeps what the controller reads from a pool's
// Deployment: its identity, labels and annotations, and its status. The pool
// controller never reads a Deployment's spec, which it doesn't change.
func transformPoolDeployment(obj any) (any, error) {
	deploy, ok := obj.(*appsv1.Deployment)
	if !ok {
		return obj, nil
	}
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:              deploy.Name,
			Namespace:         deploy.Namespace,
			UID:               deploy.UID,
			ResourceVersion:   deploy.ResourceVersion,
			CreationTimestamp: deploy.CreationTimestamp,
			DeletionTimestamp: deploy.DeletionTimestamp,
			Labels:            deploy.Labels,
			Annotations:       deploy.Annotations,
		},
		Status: deploy.Status,
	}, nil
}

// enqueueWorkflowsWaitingOnPool requeues the workflows with a TaskSet in the
// pool that no result claims, after the pool's health changed: their nodes
// carry it in their messages, or fail when the pool is unbuildable. Pool
// pod events requeue only claimants (enqueueWorkflowsClaimedBy), so this is
// how waiting nodes hear of a change, once per change.
func (wfc *WorkflowController) enqueueWorkflowsWaitingOnPool(namespace, hash string) {
	objs, err := wfc.pooledTaskSetInformer.GetIndexer().ByIndex(indexes.PoolIndex, indexes.MetaNamespaceLabelIndex(namespace, hash))
	if err != nil {
		return
	}
	results := wfc.taskResultInformer.GetStore()
	workflows := map[string]bool{}
	for _, obj := range objs {
		taskSet, err := meta.Accessor(obj)
		if err != nil {
			continue
		}
		ref, ok := dispatch.TaskSetRefFromMeta(taskSet)
		if !ok {
			continue
		}
		if _, claimed := taskSetClaim(results, namespace, ref); claimed {
			continue
		}
		if name := taskSet.GetLabels()[common.LabelKeyWorkflow]; name != "" {
			workflows[name] = true
		}
	}
	for name := range workflows {
		wfc.wfQueue.AddRateLimited(namespace + "/" + name)
	}
}
