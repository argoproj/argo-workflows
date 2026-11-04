package controller

import (
	"context"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// A pooled node's claim state is derived at each reconcile from the caches
// and recorded on its ledger only through the ledger's setters: a claim from
// the result informer (claimPooledTask), a release from the claim's absence
// there (releasePooledTasks). Informer lag is never read as a release:
// the result cache may hold a result after its deletion but never drops one
// before, so a claim the ledger recorded from it is missing from it only once
// deleted; and a deleted claim is a release only while its node hasn't ended
// and its TaskSet is cached, so neither an acked TaskSet (its node has ended)
// nor one deleted while claimed, nor one not cached yet, makes it one.

// cachedWithUID returns the object of type T that store holds under key, if
// it has uid: an object of that name with another UID was replaced.
func cachedWithUID[T metav1.Object](store cache.Store, key string, uid types.UID) (T, bool) {
	var none T
	obj, exists, err := store.GetByKey(key)
	if err != nil || !exists {
		return none, false
	}
	t, ok := obj.(T)
	if !ok || t.GetUID() != uid {
		return none, false
	}
	return t, true
}

// claimantPod returns the worker pod named name with uid from the pool pod
// cache, or nil if the cache doesn't hold it.
func (woc *wfOperationCtx) claimantPod(name string, uid types.UID) *apiv1.Pod {
	pod, _ := cachedWithUID[*apiv1.Pod](woc.controller.poolPodInformer.GetStore(), woc.wf.Namespace+"/"+name, uid)
	return pod
}

// workerStatus is the status of the pod's worker container, or nil for a
// nil pod or one without it.
func workerStatus(pod *apiv1.Pod) *apiv1.ContainerStatus {
	if pod == nil {
		return nil
	}
	for i, s := range pod.Status.ContainerStatuses {
		if s.Name == common.WorkerContainerName {
			return &pod.Status.ContainerStatuses[i]
		}
	}
	return nil
}

// workerRestartCount is the restart count of the pod's worker container, or
// 0 for a nil pod.
func workerRestartCount(pod *apiv1.Pod) int32 {
	if s := workerStatus(pod); s != nil {
		return s.RestartCount
	}
	return 0
}

// recordClaim records claim, which the ledger doesn't hold yet, on it at
// at: the claimant and the claim's creation time, on the API server's
// clock, with the restart count of the claimant's worker container as pod,
// its cached pod, has it, or 0 for a nil pod. The count may be lower than
// the claimant's own. A claim replacing one
// the ledger holds means that one was deleted first, so it is a release and
// a re-claim. It reports whether the ledger recorded a release, and whether
// it recorded the claim, which it doesn't once the task is interrupted.
func recordClaim(ledger *wfv1.PooledTaskStatus, claim dispatch.Claim, pod *apiv1.Pod, at metav1.Time) (released, claimed bool) {
	released = ledger.Release(at)
	return released, ledger.Claim(claim.UID, claim.CreatedAt, claim.WorkerPodName, claim.WorkerPodUID, workerRestartCount(pod))
}

// claimPooledTask records claim, the node's claim the ledger doesn't hold
// yet, on the node (recordClaim), with pod, its claimant as cached. An
// unended node is Running once claimed; its message, about the wait for a
// worker, is cleared. The node runs on the claimant, so its host node and
// pod IP are the claimant's; a nil pod, as for a claim first seen
// completed, leaves them.
func (woc *wfOperationCtx) claimPooledTask(ctx context.Context, node *wfv1.NodeStatus, claim dispatch.Claim, pod *apiv1.Pod) {
	log := woc.log.WithFields(logging.Fields{
		"nodeID":    node.ID,
		"claimUID":  claim.UID,
		"workerPod": claim.WorkerPodName,
		"restarts":  workerRestartCount(pod),
	})
	previous := node.PooledTask.WorkerPodName
	released, claimed := recordClaim(node.PooledTask, claim, pod, metav1.Now())
	if released {
		log.WithField("previousWorkerPod", previous).Info(ctx, "Pooled task released and claimed again")
	}
	if !claimed {
		log.Warn(ctx, "Not recording a claim on an interrupted pooled task")
		return
	}
	if !claim.Completed {
		node.Phase = wfv1.NodeRunning
		node.Message = ""
	}
	if pod != nil {
		node.HostNodeName, node.PodIP = pod.Spec.NodeName, pod.Status.PodIP
	}
	log.Info(ctx, "Pooled task claimed")
}

// releasePooledTasks records the release of every unended pooled node whose
// ledger holds a claim the result cache no longer has, while the node's
// TaskSet is cached: its worker deleted the claim to give the task back,
// which then waits for a worker, Running, with no task result in progress. It
// runs after the cached results have been applied, so a claim replacing the
// ledger's is already recorded as a re-claim.
func (woc *wfOperationCtx) releasePooledTasks(ctx context.Context) {
	released := func(node *wfv1.NodeStatus) bool {
		return !node.Phase.Completed() && node.PooledTask.State() == wfv1.PooledTaskClaimed && !woc.pooledClaimCached(node)
	}
	woc.eachPooledNode(released, func(node *wfv1.NodeStatus) {
		log := woc.log.WithFields(logging.Fields{"nodeID": node.ID, "claimUID": node.PooledTask.ClaimUID, "workerPod": node.PooledTask.WorkerPodName})
		if !woc.pooledTaskSetCached(pooledTaskSetRef(node)) {
			// Lag on a fresh TaskSet, whose arrival requeues nothing, or a
			// TaskSet deleted with the task claimed, which isn't a release.
			log.Debug(ctx, "Not releasing a pooled task whose WorkflowTaskSet isn't cached")
			woc.requeue()
			return
		}
		updated := node.DeepCopy()
		updated.PooledTask.Release(metav1.Now())
		woc.wf.Status.Nodes.Set(ctx, node.ID, *updated)
		// No task result is in progress, as before the first claim.
		woc.wf.Status.MarkTaskResultComplete(ctx, node.ID)
		woc.updated = true
		log.WithField("releases", updated.PooledTask.Releases).Info(ctx, "Pooled task released")
	})
}

// pooledClaimCached says whether the result cache holds the claim the node's
// ledger records.
func (woc *wfOperationCtx) pooledClaimCached(node *wfv1.NodeStatus) bool {
	_, ok := cachedWithUID[*wfv1.WorkflowTaskResult](woc.controller.taskResultInformer.GetStore(), woc.wf.Namespace+"/"+woc.pooledKey(node).ResultName(), node.PooledTask.ClaimUID)
	return ok
}
