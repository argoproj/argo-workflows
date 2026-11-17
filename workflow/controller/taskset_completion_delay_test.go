package controller

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	syncpkg "github.com/argoproj/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// newCompletionDelayController keeps the controller and queue requeue delays in
// sync and shuts down the queue when the test finishes.
func newCompletionDelayController(t *testing.T, requeueTime time.Duration, delay *time.Duration) *WorkflowController {
	t.Helper()
	queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: requeueTime})
	t.Cleanup(queue.ShutDown)
	return &WorkflowController{wfQueue: queue, requeueTime: requeueTime, taskSetCompletionDelay: delay}
}

// completionDelayTaskSet builds a taskset fixture with stable node IDs so tests
// can compare phase transitions for the same workflow across informer updates.
func completionDelayTaskSet(phases ...wfv1.NodePhase) *wfv1.WorkflowTaskSet {
	ts := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: "default", ResourceVersion: "1"},
		Status:     wfv1.WorkflowTaskSetStatus{Nodes: make(map[string]wfv1.NodeResult)},
	}
	for i, phase := range phases {
		ts.Status.Nodes[fmt.Sprint(i)] = wfv1.NodeResult{Phase: phase}
	}
	return ts
}

// TestTaskSetCompletionDelay checks exact queue timing across result transitions,
// preserving the normal delay unless a newly terminal result has an override.
func TestTaskSetCompletionDelay(t *testing.T) {
	for _, delay := range []*time.Duration{nil, new(time.Millisecond), new(250 * time.Millisecond), new(time.Second), new(20 * time.Second)} {
		name := "unset"
		if delay != nil {
			name = delay.String()
		}
		t.Run(name, func(t *testing.T) {
			for _, tt := range []struct {
				name     string
				before   []wfv1.NodePhase
				after    []wfv1.NodePhase
				terminal bool
			}{
				{name: "empty"},
				{name: "pending", after: []wfv1.NodePhase{wfv1.NodePending}},
				{name: "running", before: []wfv1.NodePhase{wfv1.NodePending}, after: []wfv1.NodePhase{wfv1.NodeRunning}},
				{name: "success", before: []wfv1.NodePhase{wfv1.NodeRunning}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}, terminal: true},
				{name: "failure", before: []wfv1.NodePhase{wfv1.NodeRunning}, after: []wfv1.NodePhase{wfv1.NodeFailed}, terminal: true},
				{name: "error", after: []wfv1.NodePhase{wfv1.NodeError}, terminal: true},
				{name: "skipped", after: []wfv1.NodePhase{wfv1.NodeSkipped}, terminal: true},
				{name: "omitted", after: []wfv1.NodePhase{wfv1.NodeOmitted}, terminal: true},
				{name: "first observation already succeeded", after: []wfv1.NodePhase{wfv1.NodeSucceeded}, terminal: true},
				{name: "repeated completion", before: []wfv1.NodePhase{wfv1.NodeSucceeded}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}},
				{name: "terminal to terminal", before: []wfv1.NodePhase{wfv1.NodeFailed}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}},
				{name: "status cleanup", before: []wfv1.NodePhase{wfv1.NodeSucceeded}},
				{name: "new retry node", before: []wfv1.NodePhase{wfv1.NodeFailed}, after: []wfv1.NodePhase{wfv1.NodeFailed, wfv1.NodeSucceeded}, terminal: true},
			} {
				t.Run(tt.name, func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						const normalDelay = 7 * time.Second
						controller := newCompletionDelayController(t, normalDelay, delay)
						queue := controller.wfQueue
						before, after := completionDelayTaskSet(tt.before...), completionDelayTaskSet(tt.after...)
						after.ResourceVersion = "2"
						// Metadata, spec and output-only changes must not make an already
						// terminal result eligible for the completion delay again.
						after.Annotations = map[string]string{"changed": "true"}
						after.Spec.Tasks = map[string]wfv1.Template{"next": {Name: "next"}}
						if result, ok := after.Status.Nodes["0"]; ok {
							result.Message = "updated"
							result.Outputs = &wfv1.Outputs{Result: new("result")}
							after.Status.Nodes["0"] = result
						}
						wantDelay := normalDelay
						if tt.terminal && delay != nil {
							wantDelay = min(*delay, normalDelay)
						}
						start := time.Now()
						controller.onWorkflowTaskSetUpdate(before, after)
						synctest.Wait()
						require.Zero(t, queue.Len())
						time.Sleep(wantDelay - time.Nanosecond)
						synctest.Wait()
						require.Zero(t, queue.Len(), "must not enqueue before the requested delay")
						time.Sleep(time.Nanosecond)
						synctest.Wait()
						require.Equal(t, 1, queue.Len())
						key, quit := queue.Get()
						require.False(t, quit)
						assert.Equal(t, "default/workflow", key)
						queue.Done(key)
						assert.Equal(t, wantDelay, time.Since(start))
					})
				})
			}
		})
	}
}

// TestTaskSetCompletionDelayInvalidObject verifies that updates lacking metadata
// neither panic nor enqueue immediate or delayed work, regardless of the override.
func TestTaskSetCompletionDelayInvalidObject(t *testing.T) {
	for i, delay := range []*time.Duration{nil, new(time.Millisecond), new(time.Second)} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				controller := newCompletionDelayController(t, time.Second, delay)
				queue := controller.wfQueue
				require.NotPanics(t, func() { controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), struct{}{}) })
				synctest.Wait()
				require.Zero(t, queue.Len())
				time.Sleep(time.Second)
				synctest.Wait()
				require.Zero(t, queue.Len())
			})
		})
	}
}

// TestTaskSetCompletionDelayCoalesces verifies that completion bursts share one
// queued workflow key and cannot cause concurrent processing of an in-flight key.
func TestTaskSetCompletionDelayCoalesces(t *testing.T) {
	for _, delay := range []time.Duration{time.Millisecond, time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				controller := newCompletionDelayController(t, time.Hour, &delay)
				queue := controller.wfQueue
				for range 100 {
					controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), completionDelayTaskSet(wfv1.NodeSucceeded))
				}
				synctest.Wait()
				time.Sleep(delay)
				synctest.Wait()
				require.Equal(t, 1, queue.Len(), "pending completions coalesce by workflow key")
				key, quit := queue.Get()
				require.False(t, quit)
				for range 100 {
					controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), completionDelayTaskSet(wfv1.NodeFailed))
				}
				synctest.Wait()
				time.Sleep(delay)
				synctest.Wait()
				require.Zero(t, queue.Len(), "an in-flight workflow cannot be dequeued concurrently")
				queue.Done(key)
				require.Equal(t, 1, queue.Len(), "in-flight completions produce one follow-up")
				key, quit = queue.Get()
				require.False(t, quit)
				queue.Done(key)
				require.Zero(t, queue.Len())
			})
		})
	}
}

// TestTaskSetCompletionDelayDoesNotPostponeEarlierEvent confirms that an override
// is a scheduling delay, not a minimum interval that blocks an earlier enqueue.
func TestTaskSetCompletionDelayDoesNotPostponeEarlierEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller := newCompletionDelayController(t, time.Second, new(10*time.Second))
		queue := controller.wfQueue
		running := completionDelayTaskSet(wfv1.NodeRunning)
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), running)
		synctest.Wait()
		time.Sleep(500 * time.Millisecond)
		controller.onWorkflowTaskSetUpdate(running, completionDelayTaskSet(wfv1.NodeSucceeded))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(500 * time.Millisecond)
		synctest.Wait()
		require.Equal(t, 1, queue.Len(), "the completion delay is not a minimum reconciliation interval")
	})
}

// TestTaskSetCompletionDelayPromotesDelayedWorkflow verifies that a positive delay
// moves an existing delayed entry earlier without leaving a redundant wakeup.
func TestTaskSetCompletionDelayPromotesDelayedWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller := newCompletionDelayController(t, 10*time.Second, new(250*time.Millisecond))
		queue := controller.wfQueue
		running := completionDelayTaskSet(wfv1.NodeRunning)
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), running)
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Second)
		controller.onWorkflowTaskSetUpdate(running, completionDelayTaskSet(wfv1.NodeSucceeded))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(250*time.Millisecond - time.Nanosecond)
		synctest.Wait()
		require.Zero(t, queue.Len(), "completion must wait for the positive delay")
		time.Sleep(time.Nanosecond)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
		key, quit := queue.Get()
		require.False(t, quit)
		queue.Done(key)
		require.Zero(t, queue.Len())
		time.Sleep(9 * time.Second)
		synctest.Wait()
		require.Zero(t, queue.Len(), "the original rate-limited deadline must not enqueue again")
	})
}

// TestTaskSetCompletionDelayRejectsStaleWorkflow verifies that delayed completion
// events still pass through stale-version rejection and the normal delayed retry.
func TestTaskSetCompletionDelayRejectsStaleWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller := newCompletionDelayController(t, 10*time.Second, new(250*time.Millisecond))
		queue := controller.wfQueue
		informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
		workflow := &unstructured.Unstructured{}
		workflow.SetNamespace("default")
		workflow.SetName("workflow")
		workflow.SetUID("workflow-uid")
		workflow.SetResourceVersion("1")
		require.NoError(t, informer.GetIndexer().Add(workflow))
		controller.wfInformer = informer
		controller.workflowKeyLock = syncpkg.NewKeyLock()
		controller.lastWrittenVersions = lastWrittenVersions{versions: map[types.UID]lastWrittenVersion{
			"workflow-uid": {resourceVersion: "2"},
		}}
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), completionDelayTaskSet(wfv1.NodeSucceeded))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(250 * time.Millisecond)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
		// No API clients are installed: proceeding past the stale-version guard
		// would fail rather than silently reconciling the stale workflow.
		require.True(t, controller.processNextItem(logging.TestContext(t.Context())))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(10 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len(), "stale workflow falls back to the normal delayed retry")
	})
}

// TestTaskSetCompletionDelayInformer verifies the actual informer wiring: a
// terminal status update delivered through the fake API enqueues its workflow.
func TestTaskSetCompletionDelayInformer(t *testing.T) {
	ctx := t.Context()
	initial := completionDelayTaskSet(wfv1.NodeRunning)
	client := fakewfclientset.NewClientset(initial)
	controller := newCompletionDelayController(t, time.Hour, new(time.Millisecond))
	controller.wfclientset = client
	queue := controller.wfQueue
	informer := controller.newWorkflowTaskSetInformer().Informer()
	go informer.Run(ctx.Done())
	require.True(t, cache.WaitForCacheSync(ctx.Done(), informer.HasSynced))
	completed := initial.DeepCopy()
	completed.ResourceVersion = "2"
	completed.Status.Nodes["0"] = wfv1.NodeResult{Phase: wfv1.NodeSucceeded}
	_, err := client.ArgoprojV1alpha1().WorkflowTaskSets("default").UpdateStatus(ctx, completed, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool { return queue.Len() == 1 }, 5*time.Second, time.Millisecond)
	key, quit := queue.Get()
	require.False(t, quit)
	assert.Equal(t, "default/workflow", key)
	queue.Done(key)
}
