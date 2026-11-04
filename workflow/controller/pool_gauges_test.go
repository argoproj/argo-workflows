package controller

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
)

// poolTaskSet is a pooled TaskSet of pool hash-a in namespace.
func poolTaskSet(namespace, nodeID string) *metav1.PartialObjectMetadata {
	taskSet := pooledTaskSetMeta(nodeID, "wf", map[string]string{common.LabelKeyPoolHash: "hash-a"})
	taskSet.Namespace = namespace
	return taskSet
}

// poolResult is a claim in ns by worker-pod on the TaskSet of pool hash-a
// with taskSetUID.
func poolResult(nodeID string, taskSetUID types.UID, extraLabels map[string]string) *wfv1.WorkflowTaskResult {
	result := &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
		Namespace: "ns",
		Name:      nodeID,
		UID:       types.UID("result-" + nodeID),
		Labels: map[string]string{
			common.LabelKeyWorkflow:               "wf",
			common.LabelKeyPoolName:               "my-pool",
			common.LabelKeyPoolHash:               "hash-a",
			common.LabelKeyTaskSetUID:             string(taskSetUID),
			common.LabelKeyWorkerPod:              "worker-pod",
			common.LabelKeyWorkerPodUID:           "worker-pod-uid",
			common.LabelKeyReportOutputsCompleted: "false",
		},
	}}
	maps.Copy(result.Labels, extraLabels)
	return result
}

// poolPodOf is a pool pod in ns of the pool with hash, ready or not.
func poolPodOf(hash, name string, ready bool) *apiv1.Pod {
	pod := poolPod(name, "uid-"+name, map[string]string{common.LabelKeyPoolHash: hash})
	if !ready {
		pod.Status.Phase = apiv1.PodPending
		pod.Status.Conditions = []apiv1.PodCondition{{Type: apiv1.PodReady, Status: apiv1.ConditionFalse}}
	}
	return pod
}

func storeOf(t *testing.T, objs ...any) cache.Store {
	t.Helper()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)
	for _, obj := range objs {
		require.NoError(t, store.Add(obj))
	}
	return store
}

func TestPoolGaugeSamples(t *testing.T) {
	deleting := poolPodOf("hash-a", "deleting", true)
	deleting.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	gone := poolPodOf("hash-b", "gone", true)
	gone.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	taskSets := storeOf(t,
		// Never claimed, or released: its result is gone.
		poolTaskSet("ns", "queued"),
		// A result of an earlier TaskSet of the same node doesn't claim this one.
		poolTaskSet("ns", "stale"),
		poolTaskSet("ns", "claimed"),
		// Interrupt labels the result after deleting the TaskSet, so a cached
		// TaskSet with an interrupted claim is one the cache hasn't seen go.
		poolTaskSet("ns", "interrupting"),
		// Completed, waiting for its ack.
		poolTaskSet("ns", "completed"),
		// The same hash in another namespace is another pool.
		poolTaskSet("ns2", "elsewhere"),
	)
	results := storeOf(t,
		poolResult("stale", "uid-earlier", nil),
		poolResult("claimed", "uid-claimed", nil),
		poolResult("interrupting", "uid-interrupting", map[string]string{common.LabelKeyInterrupt: "true"}),
		poolResult("completed", "uid-completed", map[string]string{common.LabelKeyReportOutputsCompleted: "true"}),
		// Interrupted: its TaskSet is gone, though its worker may still be stopping it.
		poolResult("interrupted", "uid-interrupted", map[string]string{common.LabelKeyInterrupt: "true"}),
		// A claim in another namespace with the same node ID doesn't claim ns2's TaskSet.
		poolResult("elsewhere", "uid-elsewhere", nil),
	)
	pods := storeOf(t,
		poolPodOf("hash-a", "ready-1", true),
		poolPodOf("hash-a", "ready-2", true),
		poolPodOf("hash-a", "starting", false),
		deleting,
		// A pool whose only pod is being deleted has gone.
		gone,
		// A pool whose only pod is starting has no ready pods yet.
		poolPodOf("hash-c", "starting", false),
	)

	assert.Equal(t, []metrics.PoolGaugeSample{
		{Namespace: "ns", Name: "my-pool", Hash: "hash-a", Backlog: 2, Claimed: 2, Ready: 2},
		{Namespace: "ns", Name: "my-pool", Hash: "hash-c"},
		{Namespace: "ns2", Name: "my-pool", Hash: "hash-a", Backlog: 1},
	}, poolGaugeSamples(taskSets, results, pods, nil))
}

// poolDeploymentOf is the cached Deployment of the pool my-pool with hash in
// namespace.
func poolDeploymentOf(namespace, hash string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Namespace: namespace,
		Name:      "my-pool-" + hash,
		Labels:    map[string]string{common.LabelKeyPoolName: "my-pool", common.LabelKeyPoolHash: hash},
	}}
}

func TestPoolGaugeSamplesFromDeployments(t *testing.T) {
	taskSets := storeOf(t,
		poolTaskSet("ns", "queued"),
		poolTaskSet("ns", "claimed"),
		// No Deployment in ns2: its pool can't be created, but its backlog
		// is still reported.
		poolTaskSet("ns2", "waiting"),
	)
	results := storeOf(t, poolResult("claimed", "uid-claimed", nil))
	pods := storeOf(t,
		poolPodOf("hash-a", "ready", true),
		// A pod left behind by a deleted pool.
		poolPodOf("hash-c", "leftover", true),
	)
	deployments := storeOf(t,
		poolDeploymentOf("ns", "hash-a"),
		// An idle pool, within its retention.
		poolDeploymentOf("ns", "hash-d"),
	)

	assert.Equal(t, []metrics.PoolGaugeSample{
		{Namespace: "ns", Name: "my-pool", Hash: "hash-a", Backlog: 1, Claimed: 1, Ready: 1},
		{Namespace: "ns", Name: "my-pool", Hash: "hash-d"},
		{Namespace: "ns2", Name: "my-pool", Hash: "hash-a", Backlog: 1},
	}, poolGaugeSamples(taskSets, results, pods, deployments))
}

func TestPoolGaugeSamplesNoPools(t *testing.T) {
	assert.Empty(t, poolGaugeSamples(storeOf(t), storeOf(t), storeOf(t), nil))
	assert.Empty(t, poolGaugeSamples(storeOf(t), storeOf(t), storeOf(t), storeOf(t)))
}

func TestPoolGaugesFromInformers(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	queued := &wfv1.WorkflowTaskSet{ObjectMeta: poolTaskSet("ns", "queued").ObjectMeta}
	claimed := &wfv1.WorkflowTaskSet{ObjectMeta: poolTaskSet("ns", "claimed").ObjectMeta}
	cancel, wfc := newController(ctx, queued, claimed, poolResult("claimed", "uid-claimed", nil))
	defer cancel()
	_, err := wfc.kubeclientset.CoreV1().Pods("ns").Create(ctx, poolPodOf("hash-a", "ready", true), metav1.CreateOptions{})
	require.NoError(t, err)

	attribs := attribute.NewSet(
		attribute.String(telemetry.AttribPoolNamespace, "ns"),
		attribute.String(telemetry.AttribPoolName, "my-pool"),
		attribute.String(telemetry.AttribPoolHash, "hash-a"),
	)
	gauge := func(name string) int64 {
		val, err := testExporter.GetInt64GaugeValue(ctx, name, &attribs)
		if err != nil {
			return -1
		}
		return val
	}
	require.Eventually(t, func() bool { return gauge(telemetry.InstrumentPoolReadyPods.Name()) == 1 }, 10*time.Second, 5*time.Millisecond)
	assert.Equal(t, int64(1), gauge(telemetry.InstrumentPoolBacklog.Name()))
	assert.Equal(t, int64(1), gauge(telemetry.InstrumentPoolClaimedTasks.Name()))
}
