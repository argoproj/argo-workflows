package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	v1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/metadata/metadatainformer"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	informerutil "github.com/argoproj/argo-workflows/v4/util/informer"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/indexes"
)

// The informers here cache the cluster side of hot worker pools: pooled
// TaskSets (metadata only) and pool pods. They run whether or not hotWorkers
// is enabled, because ingest, liveness and cancellation of pooled nodes always
// run; they need no RBAC beyond what the controller has for agent TaskSets and
// workflow pods. Both select this controller instance's objects only.

var (
	poolHashReq, _   = labels.NewRequirement(common.LabelKeyPoolHash, selection.Exists, nil)
	noPoolHashReq, _ = labels.NewRequirement(common.LabelKeyPoolHash, selection.DoesNotExist, nil)
	noWorkflowReq, _ = labels.NewRequirement(common.LabelKeyWorkflow, selection.DoesNotExist, nil)
)

// poolSelector selects this instance's pooled TaskSets, those with a pool
// hash, which every pooled TaskSet has and no agent TaskSet has, and its
// pool Deployments.
func (wfc *WorkflowController) poolSelector() string {
	return labels.NewSelector().Add(*poolHashReq, wfc.instanceIDReq()).String()
}

// poolPodSelector selects this instance's pool pods. Worker pods never carry
// the workflow label, so excluding it keeps workflow pods out even if a user
// labels one with a pool hash.
func (wfc *WorkflowController) poolPodSelector() string {
	return labels.NewSelector().Add(*poolHashReq, *noWorkflowReq, wfc.instanceIDReq()).String()
}

func pooledTaskSetIndexers() cache.Indexers {
	return cache.Indexers{
		indexes.WorkflowIndex: indexes.MetaWorkflowIndexFunc,
		indexes.PoolIndex:     indexes.MetaNamespaceLabelIndexFunc(common.LabelKeyPoolHash),
	}
}

// newPooledTaskSetInformer caches the metadata of this instance's pooled
// TaskSets. Their specs hold the whole template and pool spec, which the
// controller reads live when it needs them.
func (wfc *WorkflowController) newPooledTaskSetInformer(ctx context.Context) cache.SharedIndexInformer {
	selector := wfc.poolSelector()
	logging.RequireLoggerFromContext(ctx).WithField("labelSelector", selector).Info(ctx, "Watching pooled task sets")
	// resyncPeriod 0: only a deletion is news, and a resync replays no
	// deletions. The workflow informer's resync is the sweep for pooled work.
	informer := metadatainformer.NewFilteredMetadataInformer(
		wfc.metadataInterface,
		wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowTaskSetPlural),
		wfc.GetManagedNamespace(),
		0,
		pooledTaskSetIndexers(),
		func(options *metav1.ListOptions) {
			options.LabelSelector = selector
		},
	).Informer()
	wfc.configurePooledTaskSetInformer(informer)
	return informer
}

// configurePooledTaskSetInformer sets the transform and handlers of a pooled
// TaskSet informer. A pooled TaskSet is written once and never updated, so
// only its deletion is news to its workflow.
func (wfc *WorkflowController) configurePooledTaskSetInformer(informer cache.SharedIndexInformer) {
	//nolint:errcheck // the error only happens if the informer was already started, and it hasn't been
	informer.SetTransform(informerutil.StripManagedFields)
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: wfc.enqueueWorkflowFromLabel,
	})
}

// enqueueWorkflowFromLabel requeues the workflow named by an object's
// workflow label.
func (wfc *WorkflowController) enqueueWorkflowFromLabel(obj any) {
	m, err := meta.Accessor(informerutil.Untombstone(obj))
	if err != nil {
		return
	}
	if name := m.GetLabels()[common.LabelKeyWorkflow]; name != "" {
		wfc.wfQueue.AddRateLimited(m.GetNamespace() + "/" + name)
	}
}

// newPoolPodInformer caches this instance's pool pods, trimmed by
// transformPoolPod.
func (wfc *WorkflowController) newPoolPodInformer(ctx context.Context) cache.SharedIndexInformer {
	selector := wfc.poolPodSelector()
	logging.RequireLoggerFromContext(ctx).WithField("labelSelector", selector).Info(ctx, "Watching pool pods")
	// resyncPeriod 0: resyncs replay the store as same-resource-version
	// updates, which poolPodHandlers drops. The workflow informer's resync is
	// the sweep for pooled work.
	informer := v1.NewFilteredPodInformer(
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
	informer.SetTransform(transformPoolPod)
	//nolint:errcheck // the error only happens if the informer was stopped, and it hasn't even started (https://github.com/kubernetes/client-go/blob/46588f2726fa3e25b1704d6418190f424f95a990/tools/cache/shared_informer.go#L580)
	informer.AddEventHandler(wfc.poolPodHandlers())
	return informer
}

// poolPodHandlers requeue the workflows a pool pod's change can matter to.
func (wfc *WorkflowController) poolPodHandlers() cache.ResourceEventHandlerFuncs {
	return informerutil.ChangeHandlers(wfc.enqueueWorkflowsClaimedBy)
}

// transformPoolPod keeps what the controller reads from a pool pod: its
// identity, labels and owners, the node it runs on, and its status (phase,
// conditions, pod IP and container statuses).
func transformPoolPod(obj any) (any, error) {
	pod, ok := obj.(*apiv1.Pod)
	if !ok {
		return obj, nil
	}
	return &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:                       pod.Name,
			Namespace:                  pod.Namespace,
			UID:                        pod.UID,
			ResourceVersion:            pod.ResourceVersion,
			CreationTimestamp:          pod.CreationTimestamp,
			DeletionTimestamp:          pod.DeletionTimestamp,
			DeletionGracePeriodSeconds: pod.DeletionGracePeriodSeconds,
			Labels:                     pod.Labels,
			OwnerReferences:            pod.OwnerReferences,
		},
		Spec:   apiv1.PodSpec{NodeName: pod.Spec.NodeName},
		Status: pod.Status,
	}, nil
}

// enqueueWorkflowsClaimedBy requeues the workflows with an incomplete claim
// by the pool pod, found from the results' worker-pod-UID labels. Pool pods
// carry no workflow label, and a completed claim no longer depends on its
// pod.
func (wfc *WorkflowController) enqueueWorkflowsClaimedBy(obj any) {
	pod, err := meta.Accessor(informerutil.Untombstone(obj))
	if err != nil {
		return
	}
	objs, err := wfc.taskResultInformer.GetIndexer().ByIndex(indexes.WorkerPodUIDIndex, indexes.MetaNamespaceLabelIndex(pod.GetNamespace(), string(pod.GetUID())))
	if err != nil {
		return
	}
	for _, obj := range objs {
		result := obj.(*wfv1.WorkflowTaskResult)
		if result.Labels[common.LabelKeyReportOutputsCompleted] == "true" {
			continue
		}
		wfc.enqueueWorkflowFromLabel(result)
	}
}
