package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"testing/synctest"
	"time"

	syncpkg "github.com/argoproj/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	fakekube "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

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
	for _, delay := range []*time.Duration{nil, new(time.Duration(0)), new(250 * time.Millisecond), new(time.Second), new(20 * time.Second)} {
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
						queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: normalDelay})
						defer queue.ShutDown()
						controller := &WorkflowController{wfQueue: queue, taskSetCompletionDelay: delay}
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
							wantDelay = *delay
						}
						start := time.Now()
						controller.onWorkflowTaskSetUpdate(before, after)
						synctest.Wait()
						if wantDelay > 0 {
							require.Zero(t, queue.Len())
							time.Sleep(wantDelay - time.Nanosecond)
							synctest.Wait()
							require.Zero(t, queue.Len(), "must not enqueue before the requested delay")
							time.Sleep(time.Nanosecond)
							synctest.Wait()
						}
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

// TestTaskSetCompletionDelayFromEnv exercises the real constructor's optional
// integer-millisecond parsing, overflow rejection and independent normal requeue delay.
func TestTaskSetCompletionDelayFromEnv(t *testing.T) {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_ENDPOINT", "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT"} {
		t.Setenv(key, "")
	}
	t.Setenv("DEFAULT_REQUEUE_TIME", "7s")
	meterProvider, tracerProvider := otel.GetMeterProvider(), otel.GetTracerProvider()
	t.Cleanup(func() {
		otel.SetMeterProvider(meterProvider)
		otel.SetTracerProvider(tracerProvider)
	})
	for _, tt := range []struct {
		name    string
		value   string
		unset   bool
		delay   *time.Duration
		invalid bool
	}{
		{name: "unset", unset: true},
		{name: "zero", value: "0", delay: new(time.Duration(0))},
		{name: "one millisecond", value: "1", delay: new(time.Millisecond)},
		{name: "subsecond", value: "250", delay: new(250 * time.Millisecond)},
		{name: "one second", value: "1000", delay: new(time.Second)},
		{name: "longer than normal", value: "20000", delay: new(20 * time.Second)},
		{name: "maximum", value: "9223372036854", delay: new(9223372036854 * time.Millisecond)},
		{name: "empty", invalid: true},
		{name: "negative", value: "-1", invalid: true},
		{name: "malformed", value: "not-an-integer", invalid: true},
		{name: "fractional", value: "0.5", invalid: true},
		{name: "duration suffix", value: "250ms", invalid: true},
		{name: "boolean", value: "true", invalid: true},
		{name: "duration overflow", value: "9223372036855", invalid: true},
		{name: "integer overflow", value: "9223372036854775808", invalid: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const key = "ARGO_TASKSET_COMPLETION_REQUEUE_MILLISECONDS"
			t.Setenv(key, tt.value)
			if tt.unset {
				require.NoError(t, os.Unsetenv(key))
			}
			configMap := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "controller-config", Namespace: "default"},
				Data:       map[string]string{"config": "metricsConfig:\n  enabled: false\n"},
			}
			controller, err := NewWorkflowController(logging.TestContext(t.Context()),
				&rest.Config{Host: "https://kubernetes.invalid"}, fakekube.NewClientset(configMap), fakewfclientset.NewClientset(),
				"default", "default", "argoexec:test", "", "text", configMap.Name, false, false)
			if tt.invalid {
				require.ErrorContains(t, err, key+" must be an integer between 0 and")
				assert.Nil(t, controller)
				return
			}
			require.NoError(t, err)
			t.Cleanup(func() {
				controller.wfQueue.ShutDown()
				controller.wfArchiveQueue.ShutDown()
				assert.NoError(t, controller.ShutdownTracing(context.Background()))
				assert.NoError(t, controller.ShutdownMetrics(context.Background()))
			})
			assert.Equal(t, tt.delay, controller.taskSetCompletionDelay)
			assert.Equal(t, 7*time.Second, controller.requeueTime, "completion delay must not change the normal delay")
		})
	}
}

// TestTaskSetCompletionDelayInvalidObject verifies that updates lacking metadata
// neither panic nor enqueue immediate or delayed work, regardless of the override.
func TestTaskSetCompletionDelayInvalidObject(t *testing.T) {
	for i, delay := range []*time.Duration{nil, new(time.Duration(0)), new(time.Second)} {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Second})
				defer queue.ShutDown()
				controller := &WorkflowController{wfQueue: queue, taskSetCompletionDelay: delay}
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
	for _, delay := range []time.Duration{0, time.Second} {
		t.Run(delay.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Hour})
				defer queue.ShutDown()
				controller := &WorkflowController{wfQueue: queue, taskSetCompletionDelay: &delay}
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
		queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Second})
		defer queue.ShutDown()
		controller := &WorkflowController{wfQueue: queue, taskSetCompletionDelay: new(10 * time.Second)}
		running := completionDelayTaskSet(wfv1.NodeRunning)
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), running)
		controller.onWorkflowTaskSetUpdate(running, completionDelayTaskSet(wfv1.NodeSucceeded))
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len(), "the completion delay is not a minimum reconciliation interval")
	})
}

// TestTaskSetCompletionDelayPromotesDelayedWorkflow verifies that a zero override
// enqueues immediately without cancelling an already scheduled delayed wakeup.
func TestTaskSetCompletionDelayPromotesDelayedWorkflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: 10 * time.Second})
		defer queue.ShutDown()
		controller := &WorkflowController{wfQueue: queue, taskSetCompletionDelay: new(time.Duration(0))}
		running := completionDelayTaskSet(wfv1.NodeRunning)
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), running)
		synctest.Wait()
		require.Zero(t, queue.Len())
		time.Sleep(time.Second)
		controller.onWorkflowTaskSetUpdate(running, completionDelayTaskSet(wfv1.NodeSucceeded))
		require.Equal(t, 1, queue.Len())
		key, quit := queue.Get()
		require.False(t, quit)
		queue.Done(key)
		require.Zero(t, queue.Len())
		// Immediate enqueue does not cancel an existing delayed enqueue.
		time.Sleep(9 * time.Second)
		synctest.Wait()
		require.Equal(t, 1, queue.Len())
	})
}

// TestTaskSetCompletionDelayRejectsStaleWorkflow verifies that immediate completion
// events still pass through stale-version rejection and the normal delayed retry.
func TestTaskSetCompletionDelayRejectsStaleWorkflow(t *testing.T) {
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
			wfQueue: queue, wfInformer: informer, taskSetCompletionDelay: new(time.Duration(0)),
			workflowKeyLock: syncpkg.NewKeyLock(),
			lastWrittenVersions: lastWrittenVersions{versions: map[types.UID]lastWrittenVersion{
				"workflow-uid": {resourceVersion: "2"},
			}},
		}
		controller.onWorkflowTaskSetUpdate(completionDelayTaskSet(), completionDelayTaskSet(wfv1.NodeSucceeded))
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
	queue := workqueue.NewTypedRateLimitingQueue(&fixedItemIntervalRateLimiter{requeueTime: time.Hour})
	defer queue.ShutDown()
	controller := &WorkflowController{wfclientset: client, wfQueue: queue, taskSetCompletionDelay: new(time.Duration(0))}
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
