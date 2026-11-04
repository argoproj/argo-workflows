package controller

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/controller/entrypoint"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
	"github.com/argoproj/argo-workflows/v4/workflow/sync"
)

// pooledDispatchWf runs a pooled container step beside a plain one. my-image
// is in the test controller's image index, so no entrypoint lookup reaches a
// registry.
const pooledDispatchWf = `
metadata:
  name: pooled-dispatch
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
      - name: b
        template: plain
  - name: work
    pool: {}
    container:
      image: my-image
  - name: plain
    container:
      image: my-image
`

const (
	pooledStep = "pooled-dispatch[0].a"
	plainStep  = "pooled-dispatch[0].b"
)

// newPooledDispatchController returns a controller whose dispatcher is a
// dispatch.Fake, with hot workers enabled or not, holding objs.
func newPooledDispatchController(t *testing.T, enabled bool, objs ...any) (context.Context, *WorkflowController, *dispatch.Fake) {
	t.Helper()
	ctx := logging.TestContext(t.Context())
	fake := dispatch.NewFake()
	cancel, controller := newController(ctx, append(objs, func(wfc *WorkflowController) {
		wfc.dispatcher = fake.Dispatcher()
		wfc.Config.HotWorkers = &config.HotWorkersConfig{Enabled: enabled}
	})...)
	t.Cleanup(cancel)
	return ctx, controller, fake
}

// reconcilePooled runs one operation on the workflow as stored and returns
// its context.
func reconcilePooled(ctx context.Context, t *testing.T, controller *WorkflowController, wf *wfv1.Workflow) *wfOperationCtx {
	t.Helper()
	stored, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, stored, controller)
	woc.operate(ctx)
	return woc
}

func mustNode(t *testing.T, woc *wfOperationCtx, name string) *wfv1.NodeStatus {
	t.Helper()
	node, err := woc.wf.GetNodeByName(name)
	require.NoError(t, err)
	return node
}

// countOps makes fake count the calls made through it.
func countOps(fake *dispatch.Fake) *atomic.Int32 {
	var n atomic.Int32
	fake.SetFault(func(dispatch.Op) error {
		n.Add(1)
		return nil
	})
	return &n
}

// TestPooledDispatchFlagOff checks that with hot workers disabled a pooled node
// fails with a message, and the rest of the workflow runs as before.
func TestPooledDispatchFlagOff(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, false, wf)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Contains(t, node.Message, "hotWorkers.enabled")
	assert.Nil(t, node.PooledTask)
	assert.Empty(t, fake.TaskSets(wf.Namespace))
	assert.Equal(t, []string{woc.getPodName(plainStep, "plain")}, podNames(ctx, woc), "the plain step gets its pod")
}

// TestPooledDispatchFollowsLiveConfig checks that turning the flag on starts
// dispatch with the next reconcile, without a restart.
func TestPooledDispatchFollowsLiveConfig(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	later := wf.DeepCopy()
	later.Name, later.UID = "pooled-later", "later-uid"
	ctx, controller, fake := newPooledDispatchController(t, false, wf, later)

	woc := reconcilePooled(ctx, t, controller, wf)
	require.Equal(t, wfv1.NodeError, mustNode(t, woc, pooledStep).Phase)

	controller.Config.HotWorkers.Enabled = true
	woc = reconcilePooled(ctx, t, controller, later)
	node := mustNode(t, woc, "pooled-later[0].a")
	assert.Equal(t, wfv1.NodePending, node.Phase)
	require.NotNil(t, node.PooledTask)
	assert.Len(t, fake.TaskSets(wf.Namespace), 1)
}

// TestPooledDispatchEnqueues checks that a pooled node gets one TaskSet with
// spec.pool and spec.dispatch, a ledger with the TaskSet's UID, and stays
// Pending with no pod; further reconciles make no API calls for it.
func TestPooledDispatchEnqueues(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Equal(t, wfv1.NodeTypePod, node.Type)
	taskName := woc.getPodName(pooledStep, "work")

	taskSets := fake.TaskSets(wf.Namespace)
	require.Len(t, taskSets, 1)
	ts := taskSets[0]
	assert.Equal(t, node.ID+"-task", ts.Name)
	assert.Equal(t, map[string]string{
		common.LabelKeyWorkflow: wf.Name,
		common.LabelKeyPoolName: "work",
		common.LabelKeyPoolHash: node.PooledTask.PoolHash,
	}, ts.Labels)
	require.Len(t, ts.OwnerReferences, 1)
	assert.Equal(t, wf.UID, ts.OwnerReferences[0].UID)
	require.NotNil(t, ts.Spec.Pool)
	assert.Equal(t, "work", ts.Spec.Pool.Name)
	assert.Equal(t, "my-image", ts.Spec.Pool.Template.Container.Image)
	require.NotNil(t, ts.Spec.Dispatch)
	assert.Equal(t, taskName, ts.Spec.Dispatch.TaskName)
	assert.Equal(t, []string{"my-entrypoint", "my-cmd"}, ts.Spec.Dispatch.Command, "the image's entrypoint and cmd, as a pod gets them")
	assert.Equal(t, &metav1.Duration{Duration: time.Minute}, ts.Spec.Dispatch.ProgressPatchTickDuration)
	assert.Equal(t, &metav1.Duration{Duration: 3 * time.Second}, ts.Spec.Dispatch.ProgressFileTickDuration)
	assert.False(t, ts.Spec.Dispatch.IncludeScriptOutput)
	assert.Empty(t, ts.Spec.Dispatch.InputArtifactPlugins)
	assert.Contains(t, ts.Spec.Tasks, node.ID)

	ledger := node.PooledTask
	require.NotNil(t, ledger)
	assert.Equal(t, ts.UID, ledger.TaskSetUID)
	assert.Equal(t, wfv1.PooledTaskEnqueued, ledger.State())
	assert.Equal(t, "work", ledger.PoolName)
	assert.Equal(t, taskName, ledger.TaskName)
	assert.False(t, ledger.EnqueuedAt.IsZero())

	assert.Equal(t, []string{woc.getPodName(plainStep, "plain")}, podNames(ctx, woc), "no pod for the pooled node")

	ops := countOps(fake)
	for range 2 {
		woc = reconcilePooled(ctx, t, controller, wf)
	}
	assert.Zero(t, ops.Load(), "an enqueued node makes no calls")
	assert.Equal(t, *ledger, *mustNode(t, woc, pooledStep).PooledTask)
	assert.Len(t, fake.TaskSets(wf.Namespace), 1)
}

// TestPooledDispatchAdoptsAfterLostWrite checks that when the status write that
// recorded the ledger is lost, the next reconcile adopts the same TaskSet.
func TestPooledDispatchAdoptsAfterLostWrite(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	uid := mustNode(t, woc, pooledStep).PooledTask.TaskSetUID

	stored, err := controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Get(ctx, wf.Name, metav1.GetOptions{})
	require.NoError(t, err)
	id := stored.NodeID(pooledStep)
	lost := stored.Status.Nodes[id]
	lost.PooledTask = nil
	stored.Status.Nodes[id] = lost
	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows(wf.Namespace).Update(ctx, stored, metav1.UpdateOptions{})
	require.NoError(t, err)

	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	require.NotNil(t, node.PooledTask)
	assert.Equal(t, uid, node.PooledTask.TaskSetUID)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Len(t, fake.TaskSets(wf.Namespace), 1)
}

// TestPooledDispatchAdoptsStoredSettings checks adoption: an adopted TaskSet
// was resolved by the reconcile that created it, so the ledger takes its
// pool, task name and deadlines from what is stored, not from what this
// reconcile would compute.
func TestPooledDispatchAdoptsStoredSettings(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	deadline := metav1.NewTime(time.Now().Add(time.Hour).Truncate(time.Second))
	stored, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: wf.Namespace,
			Name:      wf.NodeID(pooledStep) + "-task",
			Labels:    map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyPoolName: "stored-pool", common.LabelKeyPoolHash: "stored-hash"},
			// Created by this workflow, whose status write was lost.
			OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference(wf.Name, wf.UID)},
		},
		Spec: wfv1.WorkflowTaskSetSpec{
			Pool:     &wfv1.PoolSpec{Name: "stored-pool"},
			Dispatch: &wfv1.TaskDispatch{TaskName: "stored-task", Deadline: &deadline, ActiveDeadlineSeconds: new(int64(42))},
		},
	})
	require.NoError(t, err)

	node := mustNode(t, reconcilePooled(ctx, t, controller, wf), pooledStep)
	ledger := node.PooledTask
	require.NotNil(t, ledger, node.Message)
	assert.Equal(t, stored.UID, ledger.TaskSetUID)
	assert.Equal(t, "stored-pool", ledger.PoolName)
	assert.Equal(t, "stored-hash", ledger.PoolHash)
	assert.Equal(t, "stored-task", ledger.TaskName)
	require.NotNil(t, ledger.Deadline)
	assert.True(t, deadline.Equal(ledger.Deadline), "the template has no deadline of its own")
	assert.Equal(t, new(int64(42)), ledger.ActiveDeadlineSeconds)
	assert.Len(t, fake.TaskSets(wf.Namespace), 1)
}

// TestPooledDispatchAdoptionCountsAgainstParallelism checks that a TaskSet
// adopted after a lost status write counts against the workflow's parallelism
// in the reconcile that adopts it, which didn't count its node, as the node
// had no ledger, so no other task starts beside it.
func TestPooledDispatchAdoptionCountsAgainstParallelism(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pooled-parallel
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  parallelism: 1
  templates:
  - name: main
    steps:
    - - name: a
        template: work
      - name: b
        template: work
  - name: work
    pool: {}
    container:
      image: my-image
`)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	reconcilePooled(ctx, t, controller, wf)
	require.Len(t, fake.TaskSets(wf.Namespace), 1)
	updateStored(ctx, t, controller, wf, func(stored *wfv1.Workflow) {
		for id, node := range stored.Status.Nodes {
			node.PooledTask = nil
			stored.Status.Nodes[id] = node
		}
	})

	woc := reconcilePooled(ctx, t, controller, wf)
	assert.Len(t, fake.TaskSets(wf.Namespace), 1, "the adopted task holds the only slot")
	pooled := 0
	for _, node := range woc.wf.Status.Nodes {
		if isPooled(&node) {
			pooled++
		}
	}
	assert.Equal(t, 1, pooled)
}

// TestPooledDispatchTransientError checks a transient failure to enqueue
// leaves the node Pending with the error, and the next reconcile enqueues it
// and clears the message.
func TestPooledDispatchTransientError(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	fake.SetFault(func(op dispatch.Op) error {
		if op.Verb == "create" {
			return apierr.NewServiceUnavailable("try again")
		}
		return nil
	})
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Contains(t, node.Message, "try again")
	assert.Nil(t, node.PooledTask)

	fake.SetFault(nil)
	woc = reconcilePooled(ctx, t, controller, wf)
	node = mustNode(t, woc, pooledStep)
	require.NotNil(t, node.PooledTask)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	assert.Empty(t, node.Message)
}

// TestPooledDispatchFieldsDropped checks that a TaskSet stored without
// spec.pool, as by an old CRD, fails the node with a message saying so. The
// ledger records the TaskSet, so it is cleaned up with the workflow's others.
func TestPooledDispatchFieldsDropped(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	stripped, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.Namespace,
		Name:      wf.NodeID(pooledStep) + "-task",
		Labels:    map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyPoolName: "work", common.LabelKeyPoolHash: "hash"},
		// Created by this workflow, whose status write was lost.
		OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference(wf.Name, wf.UID)},
	}})
	require.NoError(t, err)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Contains(t, node.Message, "install the current CRDs")
	require.NotNil(t, node.PooledTask)
	assert.Equal(t, stripped.UID, node.PooledTask.TaskSetUID)
}

// TestPooledDispatchNameTaken checks that a TaskSet of the task's name that
// isn't a pooled task, such as the agent TaskSet of a workflow named like it,
// fails the node and is left untouched; the node gets no ledger, so nothing
// ever deletes that TaskSet for it.
func TestPooledDispatchNameTaken(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	name := wf.NodeID(pooledStep) + "-task"
	agent, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Namespace: wf.Namespace, Name: name}})
	require.NoError(t, err)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeError, node.Phase)
	assert.Contains(t, node.Message, "isn't a pooled task")
	assert.Nil(t, node.PooledTask)
	got, ok := fake.TaskSet(wf.Namespace, name)
	require.True(t, ok)
	assert.Equal(t, agent.UID, got.UID)
}

// TestPooledDispatchReplacesAnEarlierWorkflowsTaskSet checks that a pooled
// TaskSet of the node's name left by an earlier workflow of the same name,
// deleted with background or orphan propagation, is replaced, so its claims
// can't pass the new TaskSet's fence.
func TestPooledDispatchReplacesAnEarlierWorkflowsTaskSet(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	name := wf.NodeID(pooledStep) + "-task"
	stale, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.Namespace,
		Name:      name,
		Labels:    map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyPoolName: "work", common.LabelKeyPoolHash: "old-hash"},
		OwnerReferences: []metav1.OwnerReference{{
			APIVersion: wfv1.SchemeGroupVersion.String(), Kind: workflow.WorkflowKind, Name: wf.Name, UID: "earlier-wf-uid",
		}},
	}})
	require.NoError(t, err)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodePending, node.Phase)
	require.NotNil(t, node.PooledTask)
	got, ok := fake.TaskSet(wf.Namespace, name)
	require.True(t, ok)
	assert.NotEqual(t, stale.UID, got.UID)
	assert.Equal(t, got.UID, node.PooledTask.TaskSetUID)
	assert.True(t, dispatch.OwnedByWorkflow(got, wf.UID))
}

// TestPooledDispatchShutdown checks the shutdown guard: a stopped workflow's
// pooled node fails before anything is enqueued, as a pod's does.
func TestPooledDispatchShutdown(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	wf.Spec.Shutdown = wfv1.ShutdownStrategyStop
	ctx, controller, fake := newPooledDispatchController(t, true, wf)

	woc := reconcilePooled(ctx, t, controller, wf)
	assert.Empty(t, fake.TaskSets(wf.Namespace))
	node := mustNode(t, woc, pooledStep)
	assert.Equal(t, wfv1.NodeFailed, node.Phase)
	assert.Contains(t, node.Message, "shutdown")
	assert.Nil(t, node.PooledTask)
}

const pooledParallelWf = `
metadata:
  name: pooled-parallel
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
      - name: b
        template: work
  - name: work
    pool: {}
    container:
      image: my-image
  - name: work-mutex
    pool: {}
    synchronization:
      mutexes:
      - name: pooled
    container:
      image: my-image
`

// TestPooledDispatchConstraints checks that pooled nodes respect workflow
// parallelism and synchronization as pods do.
func TestPooledDispatchConstraints(t *testing.T) {
	t.Run("Parallelism", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledParallelWf)
		wf.Spec.Parallelism = new(int64(1))
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		for range 2 {
			reconcilePooled(ctx, t, controller, wf)
		}
		assert.Len(t, fake.TaskSets(wf.Namespace), 1)
	})
	t.Run("Mutex", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow(pooledParallelWf)
		for i := range wf.Spec.Templates[0].Steps[0].Steps {
			wf.Spec.Templates[0].Steps[0].Steps[i].Template = "work-mutex"
		}
		ctx, controller, fake := newPooledDispatchController(t, true, wf)
		controller.syncManager, _ = sync.NewLockManager(ctx, controller.kubeclientset, controller.namespace, nil, getSyncLimitFunc(ctx, controller.kubeclientset), func(string) {}, workflowExistenceFunc, false)
		for range 2 {
			reconcilePooled(ctx, t, controller, wf)
		}
		assert.Len(t, fake.TaskSets(wf.Namespace), 1)
	})
}

const pooledScriptWf = `
metadata:
  name: pooled-script
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: pooled
      - name: b
        template: plain
  - name: pooled
    pool: {}
    script:
      image: my-image
      command: [sh]
      source: echo hi
  - name: plain
    script:
      image: my-image
      command: [sh]
      source: echo hi
`

// TestPooledScriptArgs checks that a pooled script's task doesn't carry the
// pod's script staging path, which a pod still gets.
func TestPooledScriptArgs(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledScriptWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)

	node := mustNode(t, woc, "pooled-script[0].a")
	require.NotNil(t, node.PooledTask)
	taskSets := fake.TaskSets(wf.Namespace)
	require.Len(t, taskSets, 1)
	ts := taskSets[0]
	assert.Empty(t, ts.Spec.Tasks[node.ID].Script.Args)
	assert.Equal(t, "echo hi", ts.Spec.Tasks[node.ID].Script.Source)
	assert.Nil(t, ts.Spec.Dispatch.Command, "the script has a command")

	pod, err := getPod(ctx, woc, woc.getPodName("pooled-script[0].b", "plain"))
	require.NoError(t, err)
	main := findMainContainer(pod)
	require.NotNil(t, main)
	assert.Contains(t, main.Args, common.ExecutorScriptSourcePath)
}

const pooledFanOutWf = `
metadata:
  name: pooled-fan-out
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
        withItems: [1, 2]
  - name: work
    pool: {}
    container:
      image: my-image
      command: [sh, -c, "echo hi > /tmp/out"]
    outputs:
      artifacts:
      - name: out
        path: /tmp/out
`

// TestPooledFanOutArchiveKeys checks that pooled fan-out tasks get distinct
// archive keys with the default key format, those a pod for each would have;
// they share a pool, and so does the same workflow submitted again.
func TestPooledFanOutArchiveKeys(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledFanOutWf)
	again := wf.DeepCopy()
	again.Name, again.UID = "pooled-fan-out-again", "again-uid"
	ctx, controller, fake := newPooledDispatchController(t, true, wf, again)

	woc := reconcilePooled(ctx, t, controller, wf)
	keys := map[string]bool{}
	hashes := map[string]bool{}
	for _, name := range []string{"pooled-fan-out[0].a(0:1)", "pooled-fan-out[0].a(1:2)"} {
		node := mustNode(t, woc, name)
		require.NotNil(t, node.PooledTask, name)
		ts, ok := fake.TaskSet(wf.Namespace, node.ID+"-task")
		require.True(t, ok, name)
		location := ts.Spec.Tasks[node.ID].ArchiveLocation
		require.NotNil(t, location, name)
		require.NotNil(t, location.S3, name)
		assert.Equal(t, wf.Name+"/"+woc.getPodName(name, "work"), location.S3.Key, "the key a pod would have")
		keys[location.S3.Key] = true
		hashes[node.PooledTask.PoolHash] = true
	}
	assert.Len(t, keys, 2)
	assert.Len(t, hashes, 1)

	woc = reconcilePooled(ctx, t, controller, again)
	node := mustNode(t, woc, "pooled-fan-out-again[0].a(0:1)")
	require.NotNil(t, node.PooledTask)
	assert.True(t, hashes[node.PooledTask.PoolHash], "the same workflow again gives the same pool")
}

// pooledRunningWf is a workflow already running, so the controller doesn't
// validate it again: what dispatch finds is what submission couldn't see,
// as when a template only becomes unpoolable once something is merged in.
const pooledRunningWf = `
metadata:
  name: pooled-running
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  arguments:
    parameters:
    - name: pool
      value: good
  templates:
  - name: main
    steps:
    - - name: a
        template: work
  - name: work
    pool:
      name: "{{workflow.parameters.pool}}"
    container:
      image: my-image
      args: [hello]
status:
  phase: Running
  startedAt: "2026-10-15T00:00:00Z"
`

// TestPooledDispatchChecksTheMergedTemplate checks that a template that only
// becomes unpoolable once merged, or substituted, fails at dispatch with a
// message naming what is wrong.
func TestPooledDispatchChecksTheMergedTemplate(t *testing.T) {
	for _, tt := range []struct {
		name    string
		modify  func(wf *wfv1.Workflow)
		wantErr string
	}{
		{name: "Poolable"},
		{
			name: "TemplateDefaults",
			modify: func(wf *wfv1.Workflow) {
				wf.Spec.TemplateDefaults = &wfv1.Template{PodSpecPatch: `{"terminationGracePeriodSeconds": 1}`}
			},
			wantErr: "podSpecPatch",
		},
		{
			name: "PodName",
			modify: func(wf *wfv1.Workflow) {
				wf.Spec.Templates[1].Container.Args = []string{"{{pod.name}}"}
			},
			wantErr: "{{node.taskName}}",
		},
		{
			name: "SubstitutedPoolName",
			modify: func(wf *wfv1.Workflow) {
				wf.Spec.Arguments.Parameters[0].Value = wfv1.AnyStringPtr("Not_A_Label")
			},
			wantErr: "'Not_A_Label' is not a valid pool name",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			wf := wfv1.MustUnmarshalWorkflow(pooledRunningWf)
			if tt.modify != nil {
				tt.modify(wf)
			}
			ctx, controller, fake := newPooledDispatchController(t, true, wf)
			woc := reconcilePooled(ctx, t, controller, wf)
			node := mustNode(t, woc, "pooled-running[0].a")
			if tt.wantErr == "" {
				require.NotNil(t, node.PooledTask, node.Message)
				assert.Equal(t, "good", node.PooledTask.PoolName, "the pool name is substituted")
				return
			}
			assert.Equal(t, wfv1.NodeError, node.Phase)
			assert.Contains(t, node.Message, tt.wantErr)
			assert.Nil(t, node.PooledTask)
			assert.Empty(t, fake.TaskSets(wf.Namespace))
		})
	}
}

// TestPooledTaskHasNoPodName checks that a pooled task's template is
// substituted without pod.name, as it has no pod of its own, so an
// expression that reads pod fails, guarded or not (a guard covers a missing
// member, not a missing variable), while node.taskName is the task name,
// with and without a retry strategy.
func TestPooledTaskHasNoPodName(t *testing.T) {
	for _, retry := range []bool{false, true} {
		t.Run(fmt.Sprintf("retry=%v", retry), func(t *testing.T) {
			run := func(t *testing.T, arg string) (*wfOperationCtx, *dispatch.Fake, *wfv1.NodeStatus) {
				t.Helper()
				wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
				wf.Spec.Templates[1].Container.Args = []string{arg}
				nodeName := pooledStep
				if retry {
					wf.Spec.Templates[1].RetryStrategy = &wfv1.RetryStrategy{Limit: new(intstr.FromInt32(1))}
					nodeName = pooledStep + "(0)"
				}
				ctx, controller, fake := newPooledDispatchController(t, true, wf)
				woc := reconcilePooled(ctx, t, controller, wf)
				return woc, fake, mustNode(t, woc, nodeName)
			}
			t.Run("TaskName", func(t *testing.T) {
				woc, fake, node := run(t, "{{node.taskName}}")
				require.NotNil(t, node.PooledTask, node.Message)
				taskSets := fake.TaskSets(woc.wf.Namespace)
				require.Len(t, taskSets, 1)
				assert.Equal(t, []string{node.PooledTask.TaskName}, taskSets[0].Spec.Tasks[node.ID].Container.Args)
			})
			for _, arg := range []string{"{{=pod.name}}", "{{=pod.name ?? 'none'}}"} {
				t.Run(arg, func(t *testing.T) {
					woc, fake, node := run(t, arg)
					assert.Equal(t, wfv1.NodeError, node.Phase)
					assert.Contains(t, node.Message, "pod is missing")
					assert.Empty(t, fake.TaskSets(woc.wf.Namespace))
				})
			}
		})
	}
}

const pooledDeadlinesWf = `
metadata:
  name: pooled-deadlines
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  activeDeadlineSeconds: 3600
  templates:
  - name: main
    steps:
    - - name: a
        template: work
  - name: work
    pool: {}
    activeDeadlineSeconds: 60
    timeout: 10m
    pendingTimeout: 5m
    container:
      image: my-image
      command: [true]
`

// TestPooledDispatchDeadlines checks spec.dispatch's deadlines: the absolute
// deadline is the earliest of the workflow deadline and the template timeout,
// from the node's start, and activeDeadlineSeconds is the template's, which
// the worker counts from the claim. The ledger copies them. The
// pendingTimeout is the controller's alone, as it bounds the wait for a claim.
func TestPooledDispatchDeadlines(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDeadlinesWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, "pooled-deadlines[0].a")
	require.NotNil(t, node.PooledTask)
	ts, ok := fake.TaskSet(wf.Namespace, node.ID+"-task")
	require.True(t, ok)
	d := ts.Spec.Dispatch

	started := node.StartedAt.Time
	require.NotNil(t, d.Deadline)
	assert.WithinDuration(t, started.Add(10*time.Minute), d.Deadline.Time, time.Second, "the timeout is earlier than the workflow deadline")
	assert.Equal(t, new(int64(60)), d.ActiveDeadlineSeconds)
	assert.Equal(t, d.Deadline, node.PooledTask.Deadline)
	assert.Equal(t, d.ActiveDeadlineSeconds, node.PooledTask.ActiveDeadlineSeconds)
}

func TestPooledTaskDeadline(t *testing.T) {
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	late := early.Add(time.Hour)
	assert.Nil(t, pooledTaskDeadline(time.Time{}, nil))
	assert.Equal(t, early, pooledTaskDeadline(early, nil).Time)
	assert.Equal(t, early, pooledTaskDeadline(time.Time{}, &early).Time)
	assert.Equal(t, early, pooledTaskDeadline(late, &early).Time)
	assert.Equal(t, early, pooledTaskDeadline(early, &late).Time)
}

// TestPooledCommand checks a pooled task's command is resolved as a pod's.
func TestPooledCommand(t *testing.T) {
	for _, tt := range []struct {
		name string
		tmpl *wfv1.Template
		want []string
	}{
		{name: "Command", tmpl: &wfv1.Template{Container: &apiv1.Container{Image: "my-image", Command: []string{"run"}}}},
		{name: "ImageEntrypointAndCmd", tmpl: &wfv1.Template{Container: &apiv1.Container{Image: "my-image"}}, want: []string{"my-entrypoint", "my-cmd"}},
		{name: "Args", tmpl: &wfv1.Template{Container: &apiv1.Container{Image: "my-image", Args: []string{"x"}}}, want: []string{"my-entrypoint"}},
		{name: "Script", tmpl: &wfv1.Template{Script: &wfv1.ScriptTemplate{Container: apiv1.Container{Image: "my-image"}, Source: "x"}}, want: []string{"my-entrypoint"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			woc := newWoc(ctx)
			ctr := tt.tmpl.Container
			if ctr == nil {
				ctr = &tt.tmpl.Script.Container
			}
			pb := woc.newPodBuilder(ctx, "node", []apiv1.Container{*ctr}, tt.tmpl, &createWorkflowPodOpts{})
			got, err := pb.pooledCommand(ctx, *ctr, tt.tmpl)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestPoolWorkflowTemplateName checks where a pooled template's default pool
// name comes from.
func TestPoolWorkflowTemplateName(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	woc := newWoc(ctx)
	assert.Empty(t, woc.poolWorkflowTemplateName(&wfv1.WorkflowStep{Template: "t"}, "local/wf"))
	assert.Equal(t, "ref", woc.poolWorkflowTemplateName(&wfv1.WorkflowStep{TemplateRef: &wfv1.TemplateRef{Name: "ref", Template: "t"}}, "local/wf"))
	assert.Equal(t, "wt", woc.poolWorkflowTemplateName(&wfv1.WorkflowStep{Template: "t"}, "namespaced/wt"))
	assert.Equal(t, "cwt", woc.poolWorkflowTemplateName(&wfv1.WorkflowStep{Template: "t"}, "cluster/cwt"))
	woc.execWf.Spec.WorkflowTemplateRef = &wfv1.WorkflowTemplateRef{Name: "wf-ref"}
	assert.Equal(t, "wf-ref", woc.poolWorkflowTemplateName(&wfv1.WorkflowStep{Template: "t"}, "local/wf"))
}

// pluginLookupDeps looks images up in entrypoints, failing on any other.
type pluginLookupDeps struct {
	stubPodBuilderDeps
	entrypoints map[string][]string
}

func (d *pluginLookupDeps) lookupImage(_ context.Context, image string, _ entrypoint.Options) (*entrypoint.Image, error) {
	e, ok := d.entrypoints[image]
	if !ok {
		return nil, fmt.Errorf("no image %q", image)
	}
	return &entrypoint.Image{Entrypoint: e, Cmd: []string{"ignored"}}, nil
}

// TestLookupPoolPluginCommands checks that dispatch puts each artifact
// plugin's image entrypoint, and not its cmd, in spec.pool, so the pool's
// Deployment needs nothing else, that it changes the hash, and that a
// failed lookup fails dispatch as it fails a pod.
func TestLookupPoolPluginCommands(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	pb := &podBuilder{
		in:   podBuilderInputs{namespace: "ns", execWfSpec: &wfv1.WorkflowSpec{}},
		deps: &pluginLookupDeps{entrypoints: map[string][]string{"in:v1": {"/bin/in"}, "out:v1": {"/bin/out"}}},
	}
	spec := &wfv1.PoolSpec{Name: "work", ArtifactPlugins: []wfv1.PoolArtifactPlugin{{Name: "in", Image: "in:v1"}, {Name: "out", Image: "out:v1"}}}
	before, err := poolHash(spec)
	require.NoError(t, err)

	require.NoError(t, pb.lookupPoolPluginCommands(ctx, spec))
	assert.Equal(t, []string{"/bin/in"}, spec.ArtifactPlugins[0].Command)
	assert.Equal(t, []string{"/bin/out"}, spec.ArtifactPlugins[1].Command)
	after, err := poolHash(spec)
	require.NoError(t, err)
	assert.NotEqual(t, before, after, "a plugin's command is part of the pool")

	spec.ArtifactPlugins = append(spec.ArtifactPlugins, wfv1.PoolArtifactPlugin{Name: "missing", Image: "missing:v1"})
	require.ErrorContains(t, pb.lookupPoolPluginCommands(ctx, spec), `failed to look-up entrypoint/cmd for image "missing:v1"`)
}

// pooledSettingsWf runs a pooled script whose result a later step reads,
// with logs archived and artifacts through plugins.
const pooledSettingsWf = `
metadata:
  name: pooled-settings
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  archiveLogs: true
  templates:
  - name: main
    steps:
    - - name: a
        template: work
    - - name: b
        template: echo
        arguments:
          parameters:
          - name: x
            value: "{{steps.a.outputs.result}}"
  - name: work
    pool: {}
    inputs:
      artifacts:
      - name: in
        path: /in
        plugin:
          name: in-plugin
          configuration: "{}"
          key: in
    outputs:
      artifacts:
      - name: out
        path: /out
        plugin:
          name: out-plugin
          configuration: "{}"
          key: out
    script:
      image: my-image
      source: echo hi
  - name: echo
    inputs:
      parameters:
      - name: x
    container:
      image: my-image
`

// TestPooledDispatchSettings checks the per-task settings a pod gets from its
// spec and environment: in spec.dispatch, whether the script's result is
// reported, the artifact plugins its inputs need, and the trace context of
// its node's span; in the task's template, whether its logs are archived; and
// in spec.pool, the artifact plugins its outputs, logs included, need.
func TestPooledDispatchSettings(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledSettingsWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf, func(wfc *WorkflowController) {
		wfc.Config.ArtifactDrivers = []config.ArtifactDriver{{Name: "in-plugin", Image: "in-plugin:v1"}, {Name: "out-plugin", Image: "out-plugin:v1"}}
		for _, image := range []string{"in-plugin:v1", "out-plugin:v1"} {
			wfc.Config.Images[image] = config.Image{Entrypoint: []string{"/bin/plugin"}}
		}
	})
	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, "pooled-settings[0].a")
	require.NotNil(t, node.PooledTask, node.Message)
	ts, ok := fake.TaskSet(wf.Namespace, node.ID+"-task")
	require.True(t, ok)
	d := ts.Spec.Dispatch
	assert.True(t, d.IncludeScriptOutput, "a later step reads the result")
	location := ts.Spec.Tasks[node.ID].ArchiveLocation
	require.NotNil(t, location)
	assert.Equal(t, new(true), location.ArchiveLogs)
	assert.Equal(t, []wfv1.ArtifactPluginName{"in-plugin"}, d.InputArtifactPlugins)
	var plugins []wfv1.ArtifactPluginName
	for _, p := range ts.Spec.Pool.ArtifactPlugins {
		plugins = append(plugins, p.Name)
	}
	assert.Contains(t, plugins, wfv1.ArtifactPluginName("out-plugin"))
	require.Contains(t, d.TraceContext, "traceparent")
	assert.Equal(t, traceContext(controller.tracing.ContextWithNode(ctx, wf.Name, wf.Namespace, node.ID)), d.TraceContext, "the node's span's")
}
