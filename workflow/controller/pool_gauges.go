package controller

import (
	"cmp"
	"context"
	"slices"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubectl/pkg/util/podutils"

	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

// registerPoolGauges feeds the pool gauges from the pooled TaskSet, task
// result and pool pod informers, which must exist by now, and the pool
// Deployment informer if the pool controller runs.
func (wfc *WorkflowController) registerPoolGauges() error {
	taskSets := wfc.pooledTaskSetInformer.GetStore()
	results := wfc.taskResultInformer.GetStore()
	pods := wfc.poolPodInformer.GetStore()
	var deployments cache.Store
	if wfc.poolDeploymentInformer != nil {
		deployments = wfc.poolDeploymentInformer.GetStore()
	}
	return wfc.metrics.RegisterPoolGauges(func(context.Context) []metrics.PoolGaugeSample {
		return poolGaugeSamples(taskSets, results, pods, deployments)
	})
}

type poolKey struct {
	namespace string
	hash      string
}

// poolGaugeSamples reports each pool's backlog, claimed tasks and ready pods
// from the caches, sorted by namespace and hash. A pool is reported
// while it has a pooled TaskSet, so the backlog of a pool that can't be
// created stays visible, and while its Deployment exists, or, without a
// Deployment cache (the pool controller isn't running), while it has a pod
// that isn't being deleted. Pods count towards a reported pool either way.
//
// A TaskSet is backlog until a result claims it, claimed while that result is
// incomplete, and neither once it is complete. A result claims the TaskSet
// whose UID it carries, so a stale result of an earlier TaskSet leaves the
// task in the backlog, as does a release, which deletes the result. The
// controller deletes a TaskSet before it interrupts the task, so an
// interrupted task is neither, even while its worker is stopping it.
func poolGaugeSamples(taskSets, results, pods, deployments cache.Store) []metrics.PoolGaugeSample {
	samples := map[poolKey]*metrics.PoolGaugeSample{}
	keyOf := func(obj metav1.Object) poolKey {
		return poolKey{namespace: obj.GetNamespace(), hash: obj.GetLabels()[common.LabelKeyPoolHash]}
	}
	add := func(obj metav1.Object) *metrics.PoolGaugeSample {
		key := keyOf(obj)
		s := &metrics.PoolGaugeSample{Namespace: key.namespace, Name: obj.GetLabels()[common.LabelKeyPoolName], Hash: key.hash}
		samples[key] = s
		return s
	}
	if deployments != nil {
		for _, obj := range deployments.List() {
			if deploy, err := meta.Accessor(obj); err == nil {
				add(deploy)
			}
		}
	}
	// sample is the sample of the pool obj belongs to, adding the pool if
	// reports is set, or nil if the pool isn't reported.
	sample := func(obj metav1.Object, reports bool) *metrics.PoolGaugeSample {
		if s, ok := samples[keyOf(obj)]; ok || !reports {
			return s
		}
		return add(obj)
	}
	for _, obj := range taskSets.List() {
		taskSet, err := meta.Accessor(obj)
		if err != nil {
			continue
		}
		ref, ok := dispatch.TaskSetRefFromMeta(taskSet)
		if !ok {
			continue
		}
		s := sample(taskSet, true)
		claim, claimed := taskSetClaim(results, taskSet.GetNamespace(), ref)
		switch {
		case !claimed:
			s.Backlog++
		case !claim.Completed:
			s.Claimed++
		}
	}
	for _, obj := range pods.List() {
		pod, ok := obj.(*apiv1.Pod)
		if !ok || pod.DeletionTimestamp != nil {
			continue
		}
		s := sample(pod, deployments == nil)
		if s != nil && podutils.IsPodReady(pod) {
			s.Ready++
		}
	}
	out := make([]metrics.PoolGaugeSample, 0, len(samples))
	for _, s := range samples {
		out = append(out, *s)
	}
	slices.SortFunc(out, func(a, b metrics.PoolGaugeSample) int {
		return cmp.Or(cmp.Compare(a.Namespace, b.Namespace), cmp.Compare(a.Hash, b.Hash))
	})
	return out
}

// taskSetClaim returns the cached claim on a pooled TaskSet, if a result
// claims it.
func taskSetClaim(results cache.Store, namespace string, ref dispatch.TaskSetRef) (dispatch.Claim, bool) {
	key := dispatch.Key{Namespace: namespace, NodeID: ref.NodeID}
	obj, exists, err := results.GetByKey(namespace + "/" + key.ResultName())
	if err != nil || !exists {
		return dispatch.Claim{}, false
	}
	result, err := meta.Accessor(obj)
	if err != nil {
		return dispatch.Claim{}, false
	}
	claim, ok := dispatch.ClaimFromResult(result)
	if !ok || claim.TaskSetUID != ref.UID {
		return dispatch.Claim{}, false
	}
	return claim, true
}
