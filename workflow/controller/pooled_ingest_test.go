package controller

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/tracing"
)

// The node and pod IP of every worker pod observeWorkerPod makes.
const (
	workerNode  = "worker-node"
	workerPodIP = "10.0.0.7"
)

// claimPooled has a worker, whose pod the controller's pool pod informer
// holds, claim the pooled node's TaskSet in fake.
func claimPooled(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, namespace, nodeID string) dispatch.Claimed {
	t.Helper()
	return claimPooledBy(ctx, t, fake, namespace, nodeID, observeWorkerPod(ctx, t, controller, namespace, "work-0123456789-abcde", "worker-pod-uid", 0))
}

// claimPooledBy has worker claim the pooled node's TaskSet in fake.
func claimPooledBy(ctx context.Context, t *testing.T, fake *dispatch.Fake, namespace, nodeID string, worker dispatch.Worker) dispatch.Claimed {
	t.Helper()
	ts, ok := fake.TaskSet(namespace, pooledTaskSetName(nodeID))
	require.True(t, ok)
	candidate, ok := dispatch.CandidateFromTaskSet(ts)
	require.True(t, ok)
	worker.PoolHash = candidate.PoolHash
	h, err := fake.Consumer(worker).Claim(ctx, candidate)
	require.NoError(t, err)
	return h
}

// observeWorkerPod makes the controller's pool pod informer hold a running
// worker pod whose running worker container has restarted restarts times,
// on node workerNode with IP workerPodIP, and returns the pod as a worker.
func observeWorkerPod(ctx context.Context, t *testing.T, controller *WorkflowController, namespace, name string, uid types.UID, restarts int32) dispatch.Worker {
	t.Helper()
	pod := &apiv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			UID:       uid,
			Labels:    map[string]string{common.LabelKeyPoolName: "work", common.LabelKeyPoolHash: "hash"},
		},
		Spec: apiv1.PodSpec{NodeName: workerNode},
		Status: apiv1.PodStatus{
			Phase: apiv1.PodRunning,
			PodIP: workerPodIP,
			ContainerStatuses: []apiv1.ContainerStatus{
				{Name: common.MainContainerName},
				{Name: common.WorkerContainerName, RestartCount: restarts, State: apiv1.ContainerState{Running: &apiv1.ContainerStateRunning{}}},
			},
		},
	}
	pods := controller.kubeclientset.CoreV1().Pods(namespace)
	existing, err := pods.Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		pod.ResourceVersion = existing.ResourceVersion
		_, err = pods.Update(ctx, pod, metav1.UpdateOptions{})
	case apierr.IsNotFound(err):
		_, err = pods.Create(ctx, pod, metav1.CreateOptions{})
	}
	require.NoError(t, err)
	waitForInformer(ctx, controller.poolPodInformer, pod, func(obj any) bool {
		cached := obj.(*apiv1.Pod)
		return cached.UID == uid && workerRestartCount(cached) == restarts
	})
	return dispatch.Worker{Namespace: namespace, PodName: name, PodUID: uid}
}

// observeResult makes the controller's result informer hold the result as
// fake has it, as the API server's watch would deliver it.
func observeResult(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, namespace, name string) {
	t.Helper()
	r, ok := fake.Result(namespace, name)
	require.True(t, ok)
	results := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskResults(namespace)
	existing, err := results.Get(ctx, name, metav1.GetOptions{})
	switch {
	case err == nil:
		r.ResourceVersion = existing.ResourceVersion
		_, err = results.Update(ctx, r, metav1.UpdateOptions{})
	case apierr.IsNotFound(err):
		_, err = results.Create(ctx, r, metav1.CreateOptions{})
	}
	require.NoError(t, err)
	waitForInformer(ctx, controller.taskResultInformer, r, func(obj any) bool {
		cached := obj.(*wfv1.WorkflowTaskResult)
		return cached.UID == r.UID && maps.Equal(cached.Labels, r.Labels) && cached.Phase == r.Phase
	})
}

// TestPooledResultUpdatesNode checks that a claim carrying the ledger's TaskSet
// UID is a task result in progress, and once complete its phase, message and
// outputs, exit code included, are the node's.
func TestPooledResultUpdatesNode(t *testing.T) {
	for _, tt := range []struct {
		name        string
		result      wfv1.NodeResult
		wantPhase   wfv1.NodePhase
		wantMessage string
	}{{
		name: "succeeded",
		result: wfv1.NodeResult{
			Phase:   wfv1.NodeSucceeded,
			Outputs: &wfv1.Outputs{Result: new("hello"), ExitCode: new("0")},
		},
		wantPhase: wfv1.NodeSucceeded,
	}, {
		name: "failed",
		result: wfv1.NodeResult{
			Phase:   wfv1.NodeFailed,
			Message: "Error (exit code 3)",
			Outputs: &wfv1.Outputs{ExitCode: new("3")},
		},
		wantPhase:   wfv1.NodeFailed,
		wantMessage: "Error (exit code 3)",
	}, {
		name:        "no final phase",
		result:      wfv1.NodeResult{Phase: wfv1.NodeRunning},
		wantPhase:   wfv1.NodeError,
		wantMessage: `pooled task completed with phase "Running"`,
	}} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			id := mustNode(t, woc, pooledStep).ID

			h := claimPooled(ctx, t, controller, fake, wf.Namespace, id)
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, pooledStep)
			assert.False(t, *node.TaskResultSynced, "a claim is a task result in progress")
			assert.False(t, node.Fulfilled())

			require.NoError(t, h.Complete(ctx, tt.result))
			observeResult(ctx, t, controller, fake, wf.Namespace, id)
			woc = reconcilePooled(ctx, t, controller, wf)
			node = mustNode(t, woc, pooledStep)
			assert.Equal(t, tt.wantPhase, node.Phase)
			assert.Equal(t, tt.wantMessage, node.Message)
			assert.Equal(t, tt.result.Outputs, node.Outputs)
			assert.True(t, *node.TaskResultSynced)
			assert.True(t, node.Fulfilled())
			assert.False(t, node.FinishedAt.IsZero())
		})
	}
}

// TestPooledResultTracesNodePhases checks that a pooled node's phase changes
// from its claim's results end its spans as a pod's do: its Pending phase span
// once it is claimed and Running, and its node span once its result
// completes, while the workflow runs on.
func TestPooledResultTracesNodePhases(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	te, exporter, err := tracing.CreateDefaultTestTracing(ctx)
	require.NoError(t, err)
	controller.tracing = te
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledStep).ID
	// ended is the status of each ended span name of the node with attrs.
	ended := func(name string, attrs ...attribute.KeyValue) []codes.Code {
		attrs = append(attrs, attribute.String(telemetry.AttribNodeID, id))
		var statuses []codes.Code
		for _, span := range exporter.GetSpansByName(name) {
			if !slices.ContainsFunc(attrs, func(a attribute.KeyValue) bool { return !slices.Contains(span.Attributes(), a) }) {
				statuses = append(statuses, span.Status().Code)
			}
		}
		return statuses
	}
	pending := attribute.String(telemetry.AttribNodePhase, string(wfv1.NodePending))
	require.Empty(t, ended("nodePhase", pending), "Pending while queued")

	h := claimPooled(ctx, t, controller, fake, wf.Namespace, id)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	reconcilePooled(ctx, t, controller, wf)
	assert.Len(t, ended("nodePhase", pending), 1, "the claim ends the Pending phase")
	assert.Empty(t, ended("node"))

	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	require.False(t, woc.wf.Status.Fulfilled(), "the plain step runs on")
	assert.Equal(t, []codes.Code{codes.Ok}, ended("node"), "the result ends the node span")
	assert.Len(t, ended("nodePhase", attribute.String(telemetry.AttribNodePhase, string(wfv1.NodeRunning))), 1)
}

// TestPooledResultIsMemoized checks that a pooled node's outputs are saved to
// its cache when its result succeeds.
func TestPooledResultIsMemoized(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pooled-memoized
  namespace: default
  uid: wf-uid
spec:
  entrypoint: work
  templates:
  - name: work
    pool: {}
    memoize:
      key: my-key
      cache:
        configMap:
          name: pooled-cache
    container:
      image: my-image
`)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, wf.Name).ID
	h := claimPooled(ctx, t, controller, fake, wf.Namespace, id)
	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{Result: new("hello"), ExitCode: new("0")}}))
	observeResult(ctx, t, controller, fake, wf.Namespace, id)

	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, wfv1.NodeSucceeded, mustNode(t, woc, wf.Name).Phase)
	cm, err := controller.kubeclientset.CoreV1().ConfigMaps(wf.Namespace).Get(ctx, "pooled-cache", metav1.GetOptions{})
	require.NoError(t, err)
	assert.Contains(t, cm.Data["my-key"], `"result":"hello"`)
}

// TestPooledResultAppliedOnceAfterAdoption checks that a claim on a node
// whose ledger write was lost is left alone, then applied, once, after the
// next dispatch adopts its TaskSet.
func TestPooledResultAppliedOnceAfterAdoption(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledStep).ID

	workflows := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace)
	stored, err := workflows.Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	lost := stored.Status.Nodes[id]
	lost.PooledTask = nil
	stored.Status.Nodes[id] = lost
	_, err = workflows.Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)
	h := claimPooled(ctx, t, controller, fake, wf.Namespace, id)
	observeResult(ctx, t, controller, fake, wf.Namespace, id)

	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	require.NotNil(t, node.PooledTask, "the TaskSet is adopted")
	assert.True(t, *node.TaskResultSynced, "the claim was left alone before adoption")
	_, kept := fake.Result(wf.Namespace, id)
	assert.True(t, kept, "and not discarded")

	woc = reconcilePooled(ctx, t, controller, wf)
	assert.False(t, *mustNode(t, woc, pooledStep).TaskResultSynced, "the claim is applied once the ledger records its TaskSet")

	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded, Outputs: &wfv1.Outputs{ExitCode: new("0")}}))
	observeResult(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	done := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeSucceeded, done.Phase)
	assert.True(t, *done.TaskResultSynced)

	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, *done, *mustNode(t, woc, pooledStep), "applying it again changes nothing")
}

// TestPooledResultWithAnotherTaskSetUIDIsDiscarded checks that a result of a
// pooled node that doesn't carry the ledger's TaskSet UID is deleted and
// changes nothing on the node, not even its TaskResultSynced.
func TestPooledResultWithAnotherTaskSetUIDIsDiscarded(t *testing.T) {
	for _, tt := range []struct {
		name   string
		labels map[string]string
		phase  wfv1.NodePhase
	}{{
		name: "claim on an older TaskSet, completed",
		labels: map[string]string{
			common.LabelKeyTaskSetUID:             "older-task-set-uid",
			common.LabelKeyWorkerPod:              "old-worker",
			common.LabelKeyWorkerPodUID:           "old-worker-uid",
			common.LabelKeyReportOutputsCompleted: "true",
		},
		phase: wfv1.NodeSucceeded,
	}, {
		name: "claim on an older TaskSet, incomplete",
		labels: map[string]string{
			common.LabelKeyTaskSetUID:             "older-task-set-uid",
			common.LabelKeyReportOutputsCompleted: "false",
		},
	}, {
		name:   "a pod's result",
		labels: map[string]string{common.LabelKeyReportOutputsCompleted: "false"},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			before := mustNode(t, woc, pooledStep)

			labels := map[string]string{common.LabelKeyWorkflow: wf.Name}
			maps.Copy(labels, tt.labels)
			_, err := fake.CreateResult(&wfv1.WorkflowTaskResult{
				ObjectMeta: metav1.ObjectMeta{Namespace: wf.Namespace, Name: before.ID, Labels: labels},
				NodeResult: wfv1.NodeResult{Phase: tt.phase, Outputs: &wfv1.Outputs{ExitCode: new("0")}},
			})
			require.NoError(t, err)
			observeResult(ctx, t, controller, fake, wf.Namespace, before.ID)

			woc = reconcilePooled(ctx, t, controller, wf)
			assert.Equal(t, *before, *mustNode(t, woc, pooledStep))
			_, exists := fake.Result(wf.Namespace, before.ID)
			assert.False(t, exists, "the result is discarded")
		})
	}
}

// TestPooledResultDiscardFailureLeavesNode checks that a failed discard
// changes nothing and requeues, and the next reconcile tries again.
func TestPooledResultDiscardFailureLeavesNode(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	before := mustNode(t, woc, pooledStep)
	_, err := fake.CreateResult(&wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.Namespace,
		Name:      before.ID,
		Labels: map[string]string{
			common.LabelKeyWorkflow:               wf.Name,
			common.LabelKeyTaskSetUID:             "older-task-set-uid",
			common.LabelKeyReportOutputsCompleted: "true",
		},
	}})
	require.NoError(t, err)
	observeResult(ctx, t, controller, fake, wf.Namespace, before.ID)
	fake.SetFault(func(op dispatch.Op) error {
		if op.Verb == "delete" && op.Resource == dispatch.ResourceResults {
			return errors.New("injected")
		}
		return nil
	})

	requeued := recordRequeues(controller)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, *before, *mustNode(t, woc, pooledStep))
	_, exists := fake.Result(wf.Namespace, before.ID)
	assert.True(t, exists)
	n, _ := requeued.take()
	assert.Equal(t, 1, n, "a failed discard requeues")

	fake.SetFault(nil)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, *before, *mustNode(t, woc, pooledStep))
	_, exists = fake.Result(wf.Namespace, before.ID)
	assert.False(t, exists)
}

// pooledClaim is a claim on the TaskSet of pooledTask's ledger.
func pooledClaim(woc *wfOperationCtx, nodeID string, completed bool) *wfv1.WorkflowTaskResult {
	return &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{
		Namespace: woc.wf.Namespace,
		Name:      nodeID,
		UID:       "claim-uid",
		Labels: map[string]string{
			common.LabelKeyWorkflow:               woc.wf.Name,
			common.LabelKeyTaskSetUID:             "task-set-uid",
			common.LabelKeyWorkerPod:              "work-0123456789-abcde",
			common.LabelKeyWorkerPodUID:           "worker-pod-uid",
			common.LabelKeyReportOutputsCompleted: map[bool]string{true: "true", false: "false"}[completed],
		},
	}}
}

// TestPooledNodeNotSyncedByStuckResultFallbacks checks that the pod path's
// fallbacks for a result that will never complete don't mark a pooled node
// synced.
func TestPooledNodeNotSyncedByStuckResultFallbacks(t *testing.T) {
	unsynced := false
	// The result fallback fires for a node whose phase has ended and whose
	// TaskResultSynced is set. An incomplete result clears that field, unless
	// the workflow tracks completion in TaskResultsCompletionStatus, as
	// workflows from older versions do, where the field stays set.
	for _, legacy := range []bool{false, true} {
		t.Run(fmt.Sprintf("result fallback, legacy completion status %v", legacy), func(t *testing.T) {
			// Ended by the controller an hour ago, its claim still incomplete
			// and its claimant still running.
			ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
				Name:             pooledChild,
				Phase:            wfv1.NodeFailed,
				FinishedAt:       metav1.NewTime(time.Now().Add(-time.Hour)),
				TaskResultSynced: new(legacy),
				PooledTask:       pooledTask(wfv1.PooledTaskInterrupted),
			})
			id := woc.wf.NodeID(pooledChild)
			if legacy {
				woc.wf.Status.TaskResultsCompletionStatus = map[string]bool{id: false}
			}
			require.NoError(t, woc.controller.taskResultInformer.GetIndexer().Add(pooledClaim(woc, id, false)))
			claimant := poolPod("work-0123456789-abcde", "worker-pod-uid", nil)
			claimant.Namespace, claimant.Status.ContainerStatuses = woc.wf.Namespace, nil
			require.NoError(t, woc.controller.poolPodInformer.GetIndexer().Add(claimant))
			woc.taskResultReconciliation(ctx)
			assert.True(t, woc.wf.Status.IsTaskResultIncomplete(id))
		})
	}
	for _, tt := range []struct {
		name   string
		status apiv1.PodStatus
	}{{
		name:   "aux container failed",
		status: apiv1.PodStatus{Phase: apiv1.PodFailed, ContainerStatuses: []apiv1.ContainerStatus{{Name: common.WaitContainerName, State: apiv1.ContainerState{Terminated: &apiv1.ContainerStateTerminated{ExitCode: 1}}}}},
	}, {
		name:   "evicted",
		status: apiv1.PodStatus{Phase: apiv1.PodFailed, Reason: "Evicted"},
	}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
				Name:             pooledChild,
				Phase:            wfv1.NodeRunning,
				TaskResultSynced: &unsynced,
				PooledTask:       pooledTask(wfv1.PooledTaskClaimed),
			})
			pod := addChildPod(t, woc, tt.status.Phase)
			pod.Status = tt.status
			require.NoError(t, woc.controller.PodController.TestingPodInformer().GetIndexer().Update(pod))
			_, err := woc.podReconciliation(ctx)
			require.NoError(t, err)
			node, err := woc.wf.Status.Nodes.Get(woc.wf.NodeID(pooledChild))
			require.NoError(t, err)
			assert.False(t, *node.TaskResultSynced)
		})
	}
}

// TestPooledResultForEndedNode checks that a node that has already ended
// takes only the arrival of its completed result from its claim: its phase,
// message and outputs stay.
func TestPooledResultForEndedNode(t *testing.T) {
	unsynced := false
	ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
		Name:             pooledChild,
		Phase:            wfv1.NodeFailed,
		Message:          "Stopped with strategy 'Terminate'",
		FinishedAt:       metav1.Now(),
		TaskResultSynced: &unsynced,
		PooledTask:       pooledTask(wfv1.PooledTaskInterrupted),
	})
	id := woc.wf.NodeID(pooledChild)
	before, err := woc.wf.Status.Nodes.Get(id)
	require.NoError(t, err)
	result := pooledClaim(woc, id, true)
	result.Phase = wfv1.NodeSucceeded
	result.Outputs = &wfv1.Outputs{ExitCode: new("0")}
	require.NoError(t, woc.controller.taskResultInformer.GetIndexer().Add(result))

	woc.taskResultReconciliation(ctx)
	node, err := woc.wf.Status.Nodes.Get(id)
	require.NoError(t, err)
	assert.True(t, *node.TaskResultSynced)
	node.TaskResultSynced = before.TaskResultSynced
	assert.Equal(t, before, node)
}

// TestPodResultsTakeThePodPath checks that a result of a node that isn't pooled
// is ingested as before.
func TestPodResultsTakeThePodPath(t *testing.T) {
	ctx, woc := newPooledNodesWoc(t, "main", wfv1.NodeStatus{
		Name:             pooledChild,
		Phase:            wfv1.NodeRunning,
		TaskResultSynced: new(true),
	})
	id := woc.wf.NodeID(pooledChild)
	result := &wfv1.WorkflowTaskResult{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: woc.wf.Namespace,
			Name:      id,
			Labels:    map[string]string{common.LabelKeyWorkflow: woc.wf.Name, common.LabelKeyReportOutputsCompleted: "false"},
		},
		NodeResult: wfv1.NodeResult{Outputs: &wfv1.Outputs{Result: new("hello")}},
	}
	assert.False(t, woc.reconcilePooledResult(ctx, result))
	require.NoError(t, woc.controller.taskResultInformer.GetIndexer().Add(result))

	woc.taskResultReconciliation(ctx)
	node, err := woc.wf.Status.Nodes.Get(id)
	require.NoError(t, err)
	assert.False(t, *node.TaskResultSynced)
	assert.Equal(t, new("hello"), node.Outputs.Result)
	assert.Equal(t, wfv1.NodeRunning, node.Phase, "a pod's result carries no phase")
}

// TestTaskResultInformerIgnoresOtherInstances checks that results labelled with
// another controller instance ID never reach the controller's cache.
func TestTaskResultInformerIgnoresOtherInstances(t *testing.T) {
	result := func(name string, extraLabels map[string]string) *wfv1.WorkflowTaskResult {
		l := map[string]string{common.LabelKeyWorkflow: "wf", common.LabelKeyTaskSetUID: "ts-" + name}
		maps.Copy(l, extraLabels)
		return &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name, Labels: l}}
	}
	mine := map[string]string{common.LabelKeyControllerInstanceID: "mine"}
	other := map[string]string{common.LabelKeyControllerInstanceID: "other"}
	for _, tt := range []struct {
		name       string
		instanceID string
		want       []string
	}{
		{name: "no instance ID", instanceID: "", want: []string{"ns/plain-1", "ns/plain-2"}},
		{name: "instance ID", instanceID: "mine", want: []string{"ns/mine-1", "ns/mine-2"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := fakewfclientset.NewClientset(result("plain-1", nil), result("mine-1", mine), result("other-1", other))
			started := filterWatches(&client.Fake, client.Tracker())
			wfc := newInformerTestController(tt.instanceID)
			wfc.wfclientset = client
			informer := wfc.newWorkflowTaskResultInformer(logging.TestContext(t.Context()))
			runInformer(t, informer)
			waitForWatch(t, started)

			// The matching ones last, so once they are cached the others
			// would have been too.
			results := client.ArgoprojV1alpha1().WorkflowTaskResults("ns")
			for _, r := range []*wfv1.WorkflowTaskResult{result("other-2", other), result("plain-2", nil), result("mine-2", mine)} {
				_, err := results.Create(t.Context(), r, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			require.Eventually(t, func() bool { return len(cacheKeys(informer)) == len(tt.want) }, 10*time.Second, 5*time.Millisecond)
			assert.ElementsMatch(t, tt.want, cacheKeys(informer))
		})
	}
}

// TestPooledResultDeleteRequeuesWorkflow checks that deleting a pooled
// task's claim requeues its workflow, and deleting a pod's result doesn't.
func TestPooledResultDeleteRequeuesWorkflow(t *testing.T) {
	claim := &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "node-1", Labels: map[string]string{
		common.LabelKeyWorkflow:   "pooled-wf",
		common.LabelKeyTaskSetUID: "ts-uid",
	}}}
	podResult := &wfv1.WorkflowTaskResult{ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: "node-2", Labels: map[string]string{
		common.LabelKeyWorkflow: "pod-wf",
	}}}
	client := fakewfclientset.NewClientset(claim, podResult)
	started := filterWatches(&client.Fake, client.Tracker())
	wfc := newInformerTestController("")
	wfc.wfclientset = client
	informer := wfc.newWorkflowTaskResultInformer(logging.TestContext(t.Context()))
	runInformer(t, informer)
	waitForWatch(t, started)
	assert.ElementsMatch(t, []string{"ns/pooled-wf", "ns/pod-wf"}, queued(t, wfc.wfQueue), "initial adds")

	results := client.ArgoprojV1alpha1().WorkflowTaskResults("ns")
	require.NoError(t, results.Delete(t.Context(), "node-2", metav1.DeleteOptions{}))
	require.NoError(t, results.Delete(t.Context(), "node-1", metav1.DeleteOptions{}))
	assert.Equal(t, []string{"ns/pooled-wf"}, queued(t, wfc.wfQueue))
	assertNothingQueued(t, wfc.wfQueue)

	t.Run("tombstone", func(t *testing.T) {
		wfc.enqueueWorkflowOfDeletedClaim(cache.DeletedFinalStateUnknown{Key: "ns/node-1", Obj: claim})
		wfc.enqueueWorkflowOfDeletedClaim(cache.DeletedFinalStateUnknown{Key: "ns/node-2", Obj: podResult})
		assert.Equal(t, []string{"ns/pooled-wf"}, queued(t, wfc.wfQueue))
	})
}
