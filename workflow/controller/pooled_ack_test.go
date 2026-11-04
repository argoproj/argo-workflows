package controller

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kwait "k8s.io/apimachinery/pkg/util/wait"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/dispatch"
)

// pooledAckWf runs a pooled step and then waits, so the workflow is still
// running once the pooled node has finished.
const pooledAckWf = `
metadata:
  name: pooled-ack
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: a
        template: work
    - - name: b
        template: wait
  - name: work
    pool: {}
    container:
      image: my-image
  - name: wait
    suspend: {}
`

const pooledAckStep = "pooled-ack[0].a"

// observeTaskSet makes the controller's pooled TaskSet informer hold the
// node's TaskSet as fake has it, as the API server's watch would deliver it.
func observeTaskSet(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, namespace, nodeID string) {
	t.Helper()
	ts, ok := fake.TaskSet(namespace, pooledTaskSetName(nodeID))
	require.True(t, ok)
	_, err := controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Create(ctx, ts, metav1.CreateOptions{})
	require.NoError(t, err)
	waitForInformer(ctx, controller.pooledTaskSetInformer, ts, func(obj any) bool {
		m, err := meta.Accessor(obj)
		return err == nil && m.GetUID() == ts.UID
	})
}

// unobserveTaskSet makes the controller's pooled TaskSet informer drop the
// node's TaskSet, as the API server's watch would on its deletion.
func unobserveTaskSet(ctx context.Context, t *testing.T, controller *WorkflowController, namespace, nodeID string) {
	t.Helper()
	name := pooledTaskSetName(nodeID)
	require.NoError(t, controller.wfclientset.ArgoprojV1alpha1().WorkflowTaskSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		_, exists, err := controller.pooledTaskSetInformer.GetStore().GetByKey(namespace + "/" + name)
		return !exists, err
	}))
}

// finishPooled has a worker claim the node's TaskSet in fake and complete it
// as succeeded, and makes the controller's result informer hold the result.
func finishPooled(ctx context.Context, t *testing.T, controller *WorkflowController, fake *dispatch.Fake, namespace, nodeID string) {
	t.Helper()
	ts, ok := fake.TaskSet(namespace, pooledTaskSetName(nodeID))
	require.True(t, ok)
	candidate, ok := dispatch.CandidateFromTaskSet(ts)
	require.True(t, ok)
	worker := dispatch.Worker{Namespace: namespace, PoolHash: candidate.PoolHash, PodName: "work-0123456789-abcde", PodUID: "worker-pod-uid"}
	h, err := fake.Consumer(worker).Claim(ctx, candidate)
	require.NoError(t, err)
	require.NoError(t, h.Complete(ctx, wfv1.NodeResult{Phase: wfv1.NodeSucceeded}))
	observeResult(ctx, t, controller, fake, namespace, nodeID)
}

// countTaskSetDeletes makes fake count the TaskSet deletes made through it,
// failing those fail says to.
func countTaskSetDeletes(fake *dispatch.Fake, fail func(name string) bool) *atomic.Int32 {
	var n atomic.Int32
	fake.SetFault(func(op dispatch.Op) error {
		if op.Verb != "delete" || op.Resource != dispatch.ResourceTaskSets {
			return nil
		}
		n.Add(1)
		if fail != nil && fail(op.Name) {
			return errors.New("transient")
		}
		return nil
	})
	return &n
}

func hasTaskSet(fake *dispatch.Fake, namespace, nodeID string) bool {
	_, ok := fake.TaskSet(namespace, pooledTaskSetName(nodeID))
	return ok
}

// TestPooledTaskSetAckedOnceFinished checks that once a pooled node is
// persisted finished its TaskSet is deleted, and that deletion changes
// nothing on the node, nor is it repeated while the cache still holds the
// TaskSet.
func TestPooledTaskSetAckedOnceFinished(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledAckStep).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)

	finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	woc = reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledAckStep)
	require.True(t, node.Fulfilled())
	assert.False(t, hasTaskSet(fake, wf.Namespace, id), "acked")
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)

	ops := countOps(fake)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Zero(t, ops.Load(), "a TaskSet acked is never deleted again, though still cached")
	assert.Equal(t, node, mustNode(t, woc, pooledAckStep), "an ack is never a release")
}

// TestPooledTaskSetAckWaitsForThePersist checks a node finished by an
// operation whose workflow write fails keeps its TaskSet, and the operation
// that persists it finished deletes it.
func TestPooledTaskSetAckWaitsForThePersist(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledAckStep).ID
	finishPooled(ctx, t, controller, fake, wf.Namespace, id)

	var failWrites atomic.Bool
	failWrites.Store(true)
	controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		if failWrites.Load() {
			return true, nil, errors.New("connection reset")
		}
		return false, nil, nil
	})
	woc = reconcilePooled(ctx, t, controller, wf)
	require.True(t, mustNode(t, woc, pooledAckStep).Fulfilled())
	assert.True(t, hasTaskSet(fake, wf.Namespace, id), "not persisted finished")

	failWrites.Store(false)
	reconcilePooled(ctx, t, controller, wf)
	assert.False(t, hasTaskSet(fake, wf.Namespace, id))
}

// TestPooledTaskSetAckRetried checks a failed ack requeues the workflow and
// is tried again while the cache holds the TaskSet, and that a node finished
// earlier whose TaskSet the cache doesn't hold, as after a restart, isn't
// acked again.
func TestPooledTaskSetAckRetried(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledAckStep).ID
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	requeued := recordRequeues(controller)

	deletes := countTaskSetDeletes(fake, func(string) bool { return true })
	reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, int32(1), deletes.Load())
	assert.True(t, hasTaskSet(fake, wf.Namespace, id))
	n, _ := requeued.take()
	assert.Equal(t, 1, n, "a failed ack requeues")

	deletes = countTaskSetDeletes(fake, nil)
	reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, int32(1), deletes.Load())
	assert.False(t, hasTaskSet(fake, wf.Namespace, id))
	n, _ = requeued.take()
	assert.Zero(t, n)

	// A restart forgets what was acked, and the cache no longer holds it.
	controller.pooledTaskSetsGone.forget(wf.UID)
	unobserveTaskSet(ctx, t, controller, wf.Namespace, id)
	deletes = countTaskSetDeletes(fake, nil)
	reconcilePooled(ctx, t, controller, wf)
	assert.Zero(t, deletes.Load())
}

// TestPooledTaskSetAckFindsItReplaced checks that an ack finding the node's
// TaskSet replaced, by an object of the same name with another UID, counts
// it as gone: the replacement is left alone, nothing is requeued, and the
// ack isn't tried again.
func TestPooledTaskSetAckFindsItReplaced(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	id := mustNode(t, woc, pooledAckStep).ID
	finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	ts, ok := fake.TaskSet(wf.Namespace, pooledTaskSetName(id))
	require.True(t, ok)
	require.True(t, fake.DeleteTaskSet(wf.Namespace, ts.Name))
	replacement := &wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{Namespace: ts.Namespace, Name: ts.Name, Labels: ts.Labels}}
	replacement, err := fake.CreateTaskSet(replacement)
	require.NoError(t, err)
	require.NotEqual(t, ts.UID, replacement.UID)
	requeued := recordRequeues(controller)

	deletes := countTaskSetDeletes(fake, nil)
	woc = reconcilePooled(ctx, t, controller, wf)
	require.True(t, mustNode(t, woc, pooledAckStep).Fulfilled())
	assert.Equal(t, int32(1), deletes.Load())
	assert.True(t, controller.pooledTaskSetsGone.has(wf.UID, ts.UID), "recorded gone")
	n, _ := requeued.take()
	assert.Zero(t, n)

	// Once cached, the replacement is still no reason to ack again.
	observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
	deletes = countTaskSetDeletes(fake, nil)
	reconcilePooled(ctx, t, controller, wf)
	assert.Zero(t, deletes.Load())
	got, ok := fake.TaskSet(wf.Namespace, ts.Name)
	require.True(t, ok)
	assert.Equal(t, replacement.UID, got.UID)
}

// TestPooledTaskSetWithDroppedFieldsAcked checks a TaskSet stored without
// spec.pool, as by an old CRD, whose node errors in the operation that adopts
// it, is deleted like any other, though not cached yet.
func TestPooledTaskSetWithDroppedFieldsAcked(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	id := wf.NodeID(pooledStep)
	_, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.Namespace,
		Name:      pooledTaskSetName(id),
		Labels:    map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyPoolName: "work", common.LabelKeyPoolHash: "hash"},
		// Created by this workflow, whose status write was lost.
		OwnerReferences: []metav1.OwnerReference{common.WorkflowOwnerReference(wf.Name, wf.UID)},
	}})
	require.NoError(t, err)

	woc := reconcilePooled(ctx, t, controller, wf)
	node := mustNode(t, woc, pooledStep)
	require.Equal(t, wfv1.NodeError, node.Phase)
	require.NotNil(t, node.PooledTask)
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase, "the plain step is still running")
	assert.False(t, hasTaskSet(fake, wf.Namespace, id))
}

// pooledPairWf runs two pooled steps side by side and is done once they are.
const pooledPairWf = `
metadata:
  name: pooled-pair
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: one
        template: work
      - name: two
        template: work
  - name: work
    pool: {}
    container:
      image: my-image
`

func workflowCompleted(woc *wfOperationCtx) bool {
	return woc.wf.Labels[common.LabelKeyCompleted] == "true"
}

// TestPooledCompletionDeletesTaskSets checks that a workflow completes only
// once its pooled TaskSets are deleted, including one known only from a
// ledger (a TaskSet without spec.pool adopted this operation) and one known
// only from the cache. A TaskSet of an earlier workflow of the same name is
// left alone.
func TestPooledCompletionDeletesTaskSets(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: pooled-done
  namespace: default
  uid: wf-uid
spec:
  entrypoint: work
  templates:
  - name: work
    pool: {}
    container:
      image: my-image
`)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	labels := map[string]string{common.LabelKeyWorkflow: wf.Name, common.LabelKeyPoolName: "work", common.LabelKeyPoolHash: "hash"}
	owner := func(uid types.UID) []metav1.OwnerReference {
		return []metav1.OwnerReference{{APIVersion: workflow.APIVersion, Kind: workflow.WorkflowKind, Name: wf.Name, UID: uid}}
	}
	id := wf.NodeID(wf.Name)
	_, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{
		Namespace: wf.Namespace, Name: pooledTaskSetName(id), Labels: labels, OwnerReferences: owner(wf.UID),
	}})
	require.NoError(t, err)
	for nodeID, uid := range map[string]types.UID{"cached-only": wf.UID, "earlier-workflow": "earlier-wf-uid"} {
		_, err := fake.CreateTaskSet(&wfv1.WorkflowTaskSet{ObjectMeta: metav1.ObjectMeta{
			Namespace: wf.Namespace, Name: pooledTaskSetName(nodeID), Labels: labels, OwnerReferences: owner(uid),
		}})
		require.NoError(t, err)
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, nodeID)
	}

	woc := reconcilePooled(ctx, t, controller, wf)
	require.Equal(t, wfv1.NodeError, mustNode(t, woc, wf.Name).Phase)
	assert.True(t, workflowCompleted(woc))
	assert.False(t, hasTaskSet(fake, wf.Namespace, id), "known from its ledger")
	assert.False(t, hasTaskSet(fake, wf.Namespace, "cached-only"))
	assert.True(t, hasTaskSet(fake, wf.Namespace, "earlier-workflow"))
}

// TestPooledCompletionAfterSizeLimit checks that a workflow whose write hits
// the size limit, and so is persisted as Error from the status it had before
// the operation, completes only once the TaskSets that operation enqueued are
// deleted too, though neither the ledgers it persists nor the cache know them.
func TestPooledCompletionAfterSizeLimit(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledDispatchWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	var writes atomic.Int32
	controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
		if writes.Add(1) == 1 {
			return true, nil, apierr.NewRequestEntityTooLargeError("etcdserver: request is too large")
		}
		return false, nil, nil
	})

	woc := reconcilePooled(ctx, t, controller, wf)
	require.Equal(t, int32(2), writes.Load(), "the size limit, then the error")
	assert.Equal(t, wfv1.WorkflowError, woc.wf.Status.Phase)
	assert.True(t, workflowCompleted(woc))
	assert.False(t, hasTaskSet(fake, wf.Namespace, wf.NodeID(pooledStep)), "the operation's TaskSet is deleted")
}

// TestPooledCompletionHeld checks that a transient failure to delete a
// pooled TaskSet holds the workflow's completion, and the next attempt
// deletes only what remains, though the cache and the ledgers still name the
// rest.
func TestPooledCompletionHeld(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledPairWf)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	woc := reconcilePooled(ctx, t, controller, wf)
	x, y := mustNode(t, woc, "pooled-pair[0].one").ID, mustNode(t, woc, "pooled-pair[0].two").ID
	for _, id := range []string{x, y} {
		observeTaskSet(ctx, t, controller, fake, wf.Namespace, id)
		finishPooled(ctx, t, controller, fake, wf.Namespace, id)
	}

	requeued := recordRequeues(controller)
	countTaskSetDeletes(fake, func(name string) bool { return name == pooledTaskSetName(x) })
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.True(t, mustNode(t, woc, "pooled-pair[0].one").Fulfilled())
	assert.Equal(t, wfv1.WorkflowRunning, woc.wf.Status.Phase)
	assert.False(t, workflowCompleted(woc))
	assert.True(t, hasTaskSet(fake, wf.Namespace, x))
	assert.False(t, hasTaskSet(fake, wf.Namespace, y))
	n, _ := requeued.take()
	assert.Equal(t, 2, n, "the failed ack and the held completion each requeue")
	assert.Contains(t, woc.wf.Status.Message, "waiting to delete pooled WorkflowTaskSets: ", "the workflow says why it can't complete")
	assert.Equal(t, 1, countEvents(controller, "WorkflowPooledTaskSetsRemain"))

	// Held again for the same reason: no second event.
	countTaskSetDeletes(fake, func(name string) bool { return name == pooledTaskSetName(x) })
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.False(t, workflowCompleted(woc))
	assert.Equal(t, 0, countEvents(controller, "WorkflowPooledTaskSetsRemain"))

	deletes := countTaskSetDeletes(fake, nil)
	woc = reconcilePooled(ctx, t, controller, wf)
	assert.Equal(t, wfv1.WorkflowSucceeded, woc.wf.Status.Phase)
	assert.True(t, workflowCompleted(woc))
	assert.Empty(t, woc.wf.Status.Message, "completing replaces the message")
	assert.Equal(t, int32(1), deletes.Load(), "only the TaskSet that remained")
	assert.Empty(t, fake.TaskSets(wf.Namespace))
}

// TestCompletionWithoutPooledTasks checks that a workflow without pooled nodes
// completes as before, with no dispatch calls.
func TestCompletionWithoutPooledTasks(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: plain
  namespace: default
  uid: wf-uid
spec:
  entrypoint: main
  templates:
  - name: main
    steps:
    - - name: skipped
        template: plain
        when: "false"
  - name: plain
    container:
      image: my-image
`)
	ctx, controller, fake := newPooledDispatchController(t, true, wf)
	ops := countOps(fake)
	woc := reconcilePooled(ctx, t, controller, wf)
	assert.True(t, workflowCompleted(woc))
	assert.Zero(t, ops.Load())
}

// TestPooledTaskSetsGoneForgottenWithWorkflow checks the record of a
// workflow's TaskSets found gone is dropped once the workflow informer stops
// holding it: when it is deleted or, through the informer's filter, when it
// completes.
func TestPooledTaskSetsGoneForgottenWithWorkflow(t *testing.T) {
	wf := wfv1.MustUnmarshalWorkflow(pooledAckWf)
	ctx, controller, _ := newPooledDispatchController(t, true, wf)
	controller.pooledTaskSetsGone.record(wf.UID, dispatch.TaskSetRef{NodeID: "node", UID: "ts-uid"})
	require.True(t, controller.pooledTaskSetsGone.has(wf.UID, "ts-uid"))

	workflows := controller.dynamicInterface.Resource(wfv1.SchemeGroupVersion.WithResource(workflow.WorkflowPlural)).Namespace(wf.Namespace)
	require.NoError(t, workflows.Delete(ctx, wf.Name, metav1.DeleteOptions{}))
	require.NoError(t, kwait.PollUntilContextTimeout(ctx, time.Millisecond, 10*time.Second, true, func(context.Context) (bool, error) {
		return !controller.pooledTaskSetsGone.has(wf.UID, "ts-uid"), nil
	}))
}

// countEvents drains the controller's recorded events and counts those with
// reason.
func countEvents(controller *WorkflowController, reason string) int {
	c := controller.eventRecorderManager.(*testEventRecorderManager).eventRecorder.Events
	n := 0
	for {
		select {
		case e := <-c:
			if strings.Contains(e, " "+reason+" ") {
				n++
			}
		default:
			return n
		}
	}
}
