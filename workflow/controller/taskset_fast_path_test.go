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

func fastPathTaskSet(phases ...wfv1.NodePhase) *wfv1.WorkflowTaskSet {
	ts := &wfv1.WorkflowTaskSet{
		ObjectMeta: metav1.ObjectMeta{Name: "workflow", Namespace: "default", ResourceVersion: "1"},
		Status:     wfv1.WorkflowTaskSetStatus{Nodes: make(map[string]wfv1.NodeResult)},
	}
	for i, phase := range phases {
		ts.Status.Nodes[fmt.Sprint(i)] = wfv1.NodeResult{Phase: phase}
	}
	return ts
}

func TestTaskSetCompletionFastPath(t *testing.T) {
	tests := []struct {
		name     string
		disabled bool
		before   []wfv1.NodePhase
		after    []wfv1.NodePhase
		fast     bool
	}{
		{name: "disabled", disabled: true, after: []wfv1.NodePhase{wfv1.NodeSucceeded}},
		{name: "empty"},
		{name: "pending", after: []wfv1.NodePhase{wfv1.NodePending}},
		{name: "running", before: []wfv1.NodePhase{wfv1.NodePending}, after: []wfv1.NodePhase{wfv1.NodeRunning}},
		{name: "success", before: []wfv1.NodePhase{wfv1.NodeRunning}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}, fast: true},
		{name: "failure", before: []wfv1.NodePhase{wfv1.NodeRunning}, after: []wfv1.NodePhase{wfv1.NodeFailed}, fast: true},
		{name: "error", after: []wfv1.NodePhase{wfv1.NodeError}, fast: true},
		{name: "skipped", after: []wfv1.NodePhase{wfv1.NodeSkipped}, fast: true},
		{name: "omitted", after: []wfv1.NodePhase{wfv1.NodeOmitted}, fast: true},
		{name: "first observation already succeeded", after: []wfv1.NodePhase{wfv1.NodeSucceeded}, fast: true},
		{name: "repeated completion", before: []wfv1.NodePhase{wfv1.NodeSucceeded}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}},
		{name: "terminal to terminal", before: []wfv1.NodePhase{wfv1.NodeFailed}, after: []wfv1.NodePhase{wfv1.NodeSucceeded}},
		{name: "status cleanup", before: []wfv1.NodePhase{wfv1.NodeSucceeded}},
		{name: "new retry node", before: []wfv1.NodePhase{wfv1.NodeFailed}, after: []wfv1.NodePhase{wfv1.NodeFailed, wfv1.NodeSucceeded}, fast: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: 10 * time.Second})
				defer queue.ShutDown()
				controller := &WorkflowController{wfQueue: queue, taskSetCompletionFastPath: !tt.disabled}
				before, after := fastPathTaskSet(tt.before...), fastPathTaskSet(tt.after...)
				after.ResourceVersion = "2"
				// Metadata, spec and output-only changes must not make an already
				// terminal result eligible for the fast path again.
				after.Annotations = map[string]string{"changed": "true"}
				after.Spec.Tasks = map[string]wfv1.Template{"next": {Name: "next"}}
				if result, ok := after.Status.Nodes["0"]; ok {
					result.Message = "updated"
					result.Outputs = &wfv1.Outputs{Result: new("result")}
					after.Status.Nodes["0"] = result
				}
				start := time.Now()
				controller.onWorkflowTaskSetUpdate(before, after)
				synctest.Wait()
				if !tt.fast {
					require.Zero(t, queue.Len(), "ordinary updates must retain the fixed delay")
					time.Sleep(10 * time.Second)
					synctest.Wait()
				}
				require.Equal(t, 1, queue.Len())
				key, quit := queue.Get()
				require.False(t, quit)
				assert.Equal(t, "default/workflow", key)
				queue.Done(key)
				t.Logf("event-to-queue delay (virtual clock): %s", time.Since(start))
			})
		})
	}
}

func TestTaskSetCompletionFastPathCoalesces(t *testing.T) {
	queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Hour})
	defer queue.ShutDown()
	controller := &WorkflowController{wfQueue: queue, taskSetCompletionFastPath: true}
	for range 100 {
		controller.onWorkflowTaskSetUpdate(fastPathTaskSet(), fastPathTaskSet(wfv1.NodeSucceeded))
	}
	require.Equal(t, 1, queue.Len(), "pending completions coalesce by workflow key")
	key, quit := queue.Get()
	require.False(t, quit)
	for range 100 {
		controller.onWorkflowTaskSetUpdate(fastPathTaskSet(), fastPathTaskSet(wfv1.NodeFailed))
	}
	require.Zero(t, queue.Len(), "an in-flight workflow cannot be dequeued concurrently")
	queue.Done(key)
	require.Equal(t, 1, queue.Len(), "in-flight completions produce one follow-up")
	key, quit = queue.Get()
	require.False(t, quit)
	queue.Done(key)
	require.Zero(t, queue.Len())
}

func TestTaskSetCompletionFastPathPromotesDelayedWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: 10 * time.Second})
		defer queue.ShutDown()
		controller := &WorkflowController{wfQueue: queue, taskSetCompletionFastPath: true}
		running := fastPathTaskSet(wfv1.NodeRunning)
		controller.onWorkflowTaskSetUpdate(fastPathTaskSet(), running)
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Second)
		controller.onWorkflowTaskSetUpdate(running, fastPathTaskSet(wfv1.NodeSucceeded))
		require.Equal(t, 1, queue.Len(), "completion bypasses an already scheduled delayed enqueue")
		key, quit := queue.Get()
		require.False(t, quit)
		queue.Done(key)
		require.Zero(t, queue.Len())
		// Add does not cancel an existing delayed enqueue. The normal worker
		// must still tolerate a redundant reconciliation at the original deadline.
		time.Sleep(9 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
	})
}

func TestTaskSetCompletionFastPathRejectsStaleWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: 10 * time.Second})
		defer queue.ShutDown()
		informer := cache.NewSharedIndexInformer(&cache.ListWatch{}, &unstructured.Unstructured{}, 0, cache.Indexers{})
		workflow := &unstructured.Unstructured{}
		workflow.SetNamespace("default")
		workflow.SetName("workflow")
		workflow.SetUID("workflow-uid")
		workflow.SetResourceVersion("1")
		require.NoError(t, informer.GetIndexer().Add(workflow))
		controller := &WorkflowController{
			wfQueue: queue, wfInformer: informer, taskSetCompletionFastPath: true,
			workflowKeyLock: syncpkg.NewKeyLock(),
			lastWrittenVersions: lastWrittenVersions{versions: map[types.UID]lastWrittenVersion{
				"workflow-uid": {resourceVersion: "2"},
			}},
		}
		controller.onWorkflowTaskSetUpdate(fastPathTaskSet(), fastPathTaskSet(wfv1.NodeSucceeded))
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

func TestTaskSetCompletionFastPathInformer(t *testing.T) {
	ctx := t.Context()
	initial := fastPathTaskSet(wfv1.NodeRunning)
	client := fakewfclientset.NewClientset(initial)
	queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Hour})
	defer queue.ShutDown()
	controller := &WorkflowController{wfclientset: client, wfQueue: queue, taskSetCompletionFastPath: true}
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
