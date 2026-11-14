package controller

import (
	"context"
	"math"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	syncpkg "github.com/argoproj/pkg/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	k8stesting "k8s.io/client-go/testing"

	"github.com/argoproj/argo-workflows/v4/persist/sqldb/mocks"
	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/argoproj/argo-workflows/v4/workflow/hydrator"
)

// TestCompletionRequeueControllerConfiguration checks startup settings and warns
// only when the completion floor exceeds the normal requeue ceiling.
func TestCompletionRequeueControllerConfiguration(t *testing.T) {
	t.Run("invalid target with scheduling disabled", func(t *testing.T) {
		t.Setenv("COMPLETED_REQUEUE_TIME_MS", "")
		t.Setenv("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT", "101")
		controller, err := NewWorkflowController(logging.TestContext(t.Context()), nil, nil, nil, "", "", "", "", "", "", false, false)
		require.ErrorContains(t, err, "COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT")
		assert.Nil(t, controller)
	})
	for _, tt := range []struct {
		name        string
		floor       string
		percent     string
		wantFloor   *time.Duration
		wantPercent int
		wantWarning bool
	}{
		{"defaults", "", "", nil, 20, false},
		{"configured", "250", "25", new(250 * time.Millisecond), 25, false},
		{"no extra gap", "1", "100", new(time.Millisecond), 100, false},
		{"floor equals ceiling", "10000", "20", new(10 * time.Second), 20, false},
		{"floor exceeds ceiling", "20000", "20", new(20 * time.Second), 20, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("COMPLETED_REQUEUE_TIME_MS", tt.floor)
			t.Setenv("COMPLETED_REQUEUE_DUTY_CYCLE_PERCENT", tt.percent)
			t.Setenv(common.EnvVarDefaultRequeueTime, "10s")
			hook := logging.NewTestHook()
			ctx, cancel := context.WithCancel(logging.WithLogger(t.Context(), logging.NewTestLogger(logging.Info, logging.Text, hook)))
			defer cancel()
			kube := fake.NewClientset(&apiv1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
				Name: "workflow-controller-configmap", Namespace: "default",
			}})
			controller, err := NewWorkflowController(ctx, &rest.Config{Host: "https://localhost"}, kube, fakewfclientset.NewClientset(),
				"default", "default", "argoexec:test", "", "text", "workflow-controller-configmap", false, false)
			require.NoError(t, err)
			defer controller.wfQueue.ShutDown()
			defer controller.wfArchiveQueue.ShutDown()
			defer func() { assert.NoError(t, controller.ShutdownMetrics(ctx)) }()
			defer func() { assert.NoError(t, controller.ShutdownTracing(ctx)) }()
			assert.Equal(t, tt.wantFloor, controller.taskSetCompletionDelay)
			assert.Equal(t, tt.wantPercent, controller.completionDutyCyclePercent)
			warnings := 0
			for _, entry := range hook.AllEntries() {
				if entry.Msg == "COMPLETED_REQUEUE_TIME_MS exceeds DEFAULT_REQUEUE_TIME; the completion delay is capped at DEFAULT_REQUEUE_TIME" {
					warnings++
					assert.Equal(t, logging.Warn, entry.Level)
				}
			}
			if tt.wantWarning {
				assert.Equal(t, 1, warnings)
			} else {
				assert.Zero(t, warnings)
			}
		})
	}
}

// TestCompletionRequeueDelay covers duty-cycle bounds, elapsed idle time, overflow
// safety, and timing isolation between workflow UIDs.
func TestCompletionRequeueDelay(t *testing.T) {
	now := time.Now()
	for _, tt := range []struct {
		name     string
		duration time.Duration
		elapsed  time.Duration
		percent  int
		floor    time.Duration
		ceiling  time.Duration
		want     time.Duration
	}{
		{"cheap", time.Millisecond, 0, 20, time.Second, 10 * time.Second, time.Second},
		{"proportional", time.Second, 0, 20, time.Millisecond, 10 * time.Second, 4 * time.Second},
		{"elapsed gap", time.Second, time.Second, 20, time.Millisecond, 10 * time.Second, 3 * time.Second},
		{"elapsed beyond gap", time.Second, 10 * time.Second, 20, time.Millisecond, 10 * time.Second, time.Millisecond},
		{"ceiling", 30 * time.Second, 0, 20, time.Millisecond, 10 * time.Second, 10 * time.Second},
		{"100 percent", 30 * time.Second, 0, 100, time.Millisecond, 10 * time.Second, time.Millisecond},
		{"1 percent", time.Second, 0, 1, time.Millisecond, 100 * time.Second, 99 * time.Second},
		{"floor capped", time.Millisecond, 0, 20, 20 * time.Second, 10 * time.Second, 10 * time.Second},
		{"overflow safe", time.Duration(math.MaxInt64), 0, 1, time.Millisecond, 10 * time.Second, 10 * time.Second},
		{"maximum ceiling", time.Duration(math.MaxInt64), 0, 1, time.Millisecond, time.Duration(math.MaxInt64), time.Duration(math.MaxInt64)},
	} {
		t.Run(tt.name, func(t *testing.T) {
			controller := &WorkflowController{
				taskSetCompletionDelay:     &tt.floor,
				requeueTime:                tt.ceiling,
				completionDutyCyclePercent: tt.percent,
				reconcileTimings: map[types.UID]workflowReconcileTiming{
					"uid": {duration: tt.duration, lastEnd: now.Add(-tt.elapsed)},
				},
			}
			assert.Equal(t, tt.want, controller.completionRequeueDelay("uid", now))
			assert.Equal(t, min(tt.floor, tt.ceiling), controller.completionRequeueDelay("new-uid", now), "same-name replacement must not inherit timing")
		})
	}
}

// TestWorkflowReconcileTiming checks duration smoothing, utilization sampling,
// and immediate timing cleanup when a workflow completes.
func TestWorkflowReconcileTiming(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, exporter := newCompletionDelayController(t, 10*time.Second, new(time.Millisecond))
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{UID: "uid"}}
	start := time.Now()
	controller.recordWorkflowReconcile(ctx, wf, start, start.Add(time.Second))
	assert.Equal(t, time.Second, controller.reconcileTimings[wf.UID].duration, "first sample seeds EWMA")
	attrs := attribute.NewSet()
	_, err := exporter.GetFloat64HistogramData(ctx, telemetry.InstrumentWorkflowReconcileUtilization.Name(), &attrs)
	require.Error(t, err, "first reconcile has no utilization interval")

	controller.recordWorkflowReconcile(ctx, wf, start.Add(5*time.Second), start.Add(8*time.Second))
	assert.Equal(t, 1400*time.Millisecond, controller.reconcileTimings[wf.UID].duration, "20%% new sample + 80%% prior EWMA")
	assert.Equal(t, start.Add(8*time.Second), controller.reconcileTimings[wf.UID].lastEnd)
	data, err := exporter.GetFloat64HistogramData(ctx, telemetry.InstrumentWorkflowReconcileUtilization.Name(), &attrs)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), data.Count)
	assert.InDelta(t, 3.0/7, data.Sum, 0.000001)
	assert.Equal(t, 0, data.Attributes.Len(), "do not create a metric series per workflow")

	wf.Labels = map[string]string{common.LabelKeyCompleted: "true"}
	controller.recordWorkflowReconcile(ctx, wf, start.Add(9*time.Second), start.Add(10*time.Second))
	assert.Empty(t, controller.reconcileTimings, "completion clears timing immediately")
}

// TestWorkflowReconcileTimingDisabled ensures the opt-out path neither allocates
// timing state nor emits utilization metrics or triggers lifecycle cleanup.
func TestWorkflowReconcileTimingDisabled(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, exporter := newCompletionDelayController(t, 10*time.Second, nil)
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{UID: "uid"}}
	start := time.Now()
	controller.recordWorkflowReconcile(ctx, wf, start, start.Add(time.Second))
	controller.recordWorkflowReconcile(ctx, wf, start.Add(5*time.Second), start.Add(6*time.Second))
	assert.Nil(t, controller.reconcileTimings, "disabled fast path must not allocate timing state")
	assert.True(t, controller.lastWrittenVersions.nextCleanup.IsZero(), "disabled timing must not trigger lifecycle bookkeeping")
	attrs := attribute.NewSet()
	_, err := exporter.GetFloat64HistogramData(ctx, telemetry.InstrumentWorkflowReconcileUtilization.Name(), &attrs)
	require.Error(t, err, "disabled fast path must not emit utilization samples")
}

// TestWorkflowReconcileTimingCleanup verifies writes preserve timing, completion
// prevents its resurrection, and periodic cleanup removes idle records.
func TestWorkflowReconcileTimingCleanup(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		controller, _ := newCompletionDelayController(t, 10*time.Second, new(time.Millisecond))
		controller.lastWrittenVersions.versions = make(map[types.UID]lastWrittenVersion)
		wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{UID: "uid", ResourceVersion: "1"}}
		ctx := logging.TestContext(t.Context())
		record := func() { controller.recordWorkflowReconcile(ctx, wf, time.Now().Add(-time.Second), time.Now()) }
		record()
		controller.recordWorkflowWrite(wf)
		assert.Equal(t, time.Second, controller.reconcileTimings[wf.UID].duration, "persisting a workflow must preserve timing")
		controller.recordWorkflowCompleted(wf)
		assert.Empty(t, controller.reconcileTimings)
		record()
		assert.Empty(t, controller.reconcileTimings, "a reconcile finishing after deletion must not resurrect timing")

		wf.UID = "replacement-uid"
		record()
		assert.Len(t, controller.reconcileTimings, 1)
		time.Sleep(2*workflowResyncPeriod + time.Nanosecond)
		wf.UID = "active-uid"
		record()
		assert.NotContains(t, controller.reconcileTimings, types.UID("replacement-uid"), "sweep idle stragglers")
		assert.Contains(t, controller.reconcileTimings, wf.UID)
	})
}

// TestWorkflowReconcileTimingConcurrent exercises timing reads, updates, and
// completion cleanup concurrently to expose races and resurrected records.
func TestWorkflowReconcileTimingConcurrent(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	controller, _ := newCompletionDelayController(t, 10*time.Second, new(time.Millisecond))
	controller.lastWrittenVersions.versions = make(map[types.UID]lastWrittenVersion)
	wf := &wfv1.Workflow{ObjectMeta: metav1.ObjectMeta{UID: "uid", ResourceVersion: "1"}}
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			for range 100 {
				now := time.Now()
				controller.recordWorkflowReconcile(ctx, wf, now.Add(-time.Millisecond), now)
				controller.completionRequeueDelay(wf.UID, now)
				controller.recordWorkflowCompleted(wf)
			}
		})
	}
	wg.Wait()
	assert.Empty(t, controller.reconcileTimings)
}

// TestTaskSetCompletionDutyCycle checks queue deadlines and metric sampling for
// completion and routine events, including disabled timing and owner lookup.
func TestTaskSetCompletionDutyCycle(t *testing.T) {
	for _, tt := range []struct {
		name       string
		floor      *time.Duration
		owner      types.UID
		kind       string
		apiVersion string
		phase      wfv1.NodePhase
		want       time.Duration
	}{
		{"completion", new(time.Millisecond), "uid", workflow.WorkflowKind, workflow.APIVersion, wfv1.NodeSucceeded, 3 * time.Second},
		{"routine", new(time.Millisecond), "uid", workflow.WorkflowKind, workflow.APIVersion, wfv1.NodeRunning, 10 * time.Second},
		{"disabled", nil, "uid", workflow.WorkflowKind, workflow.APIVersion, wfv1.NodeSucceeded, 10 * time.Second},
		{"unknown owner", new(time.Millisecond), "new-uid", workflow.WorkflowKind, workflow.APIVersion, wfv1.NodeSucceeded, time.Millisecond},
		{"missing owner API version", new(time.Millisecond), "uid", workflow.WorkflowKind, "", wfv1.NodeSucceeded, 3 * time.Second},
		{"different owner API version", new(time.Millisecond), "uid", workflow.WorkflowKind, "argoproj.io/v1", wfv1.NodeSucceeded, 3 * time.Second},
		{"different owner kind", new(time.Millisecond), "uid", "WorkflowTemplate", workflow.APIVersion, wfv1.NodeSucceeded, time.Millisecond},
	} {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx := logging.TestContext(t.Context())
				controller, exporter := newCompletionDelayController(t, 10*time.Second, tt.floor)
				controller.reconcileTimings = map[types.UID]workflowReconcileTiming{
					"uid": {duration: time.Second, lastEnd: time.Now().Add(-time.Second)},
				}
				current := completionDelayTaskSet(tt.phase)
				current.OwnerReferences = []metav1.OwnerReference{{APIVersion: tt.apiVersion, Kind: tt.kind, UID: tt.owner}}
				controller.onWorkflowTaskSetUpdate(ctx, completionDelayTaskSet(), current)
				attrs := attribute.NewSet()
				data, err := exporter.GetFloat64HistogramData(ctx, telemetry.InstrumentWorkflowCompletionRequeueDelay.Name(), &attrs)
				if tt.floor != nil && tt.phase == wfv1.NodeSucceeded {
					require.NoError(t, err)
					assert.Equal(t, uint64(1), data.Count)
					assert.InDelta(t, tt.want.Seconds(), data.Sum, 0.000001)
					assert.Zero(t, data.Attributes.Len())
				} else {
					require.Error(t, err, "routine and disabled enqueues do not emit a completion delay")
				}
				synctest.Wait()
				time.Sleep(tt.want - time.Nanosecond)
				synctest.Wait()
				assert.Zero(t, controller.wfQueue.Len())
				time.Sleep(time.Nanosecond)
				synctest.Wait()
				assert.Equal(t, 1, controller.wfQueue.Len())
			})
		})
	}
}

// A timed lock avoids synctest's non-durable sync.Mutex waits while exercising
// the same lock acquisition boundary in processNextItem.
type delayedWorkflowKeyLock struct {
	syncpkg.KeyLock
}

// Lock simulates contention with a synctest-compatible delay before acquisition.
func (l delayedWorkflowKeyLock) Lock(key string) {
	time.Sleep(5 * time.Second)
	l.KeyLock.Lock(key)
}

// TestWorkflowReconcileTimingIncludesHydration verifies duty-cycle timing includes
// hydration but not lock contention, without changing the operation-duration metric.
func TestWorkflowReconcileTimingIncludesHydration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := logging.TestContext(t.Context())
		wf := wfv1.MustUnmarshalWorkflow(`
metadata:
  name: timing
  namespace: default
  uid: timing-uid
spec:
  entrypoint: main
  templates:
    - name: main
      suspend: {}
status:
  offloadNodeStatusVersion: offloaded-version
`)
		repo := &mocks.OffloadNodeStatusRepo{}
		repo.On("Get", string(wf.UID), "offloaded-version").Run(func(mock.Arguments) {
			time.Sleep(2 * time.Second)
		}).Return(wfv1.Nodes{}, nil).Once()
		cancel, controller := newController(ctx, wf, func(controller *WorkflowController) {
			controller.taskSetCompletionDelay = new(time.Millisecond)
			controller.hydrator = hydrator.New(repo)
		})
		defer cancel()
		defer controller.wfArchiveQueue.ShutDown()
		defer func() { assert.NoError(t, controller.ShutdownMetrics(ctx)) }()
		defer func() { assert.NoError(t, controller.ShutdownTracing(ctx)) }()
		controller.wfclientset.(*fakewfclientset.Clientset).PrependReactor("update", "workflows", func(k8stesting.Action) (bool, runtime.Object, error) {
			time.Sleep(time.Second)
			return false, nil, nil
		})
		controller.workflowKeyLock = delayedWorkflowKeyLock{controller.workflowKeyLock}
		require.True(t, controller.processNextItem(ctx))
		controller.lastWrittenVersions.mutex.RLock()
		timing := controller.reconcileTimings[wf.UID]
		controller.lastWrittenVersions.mutex.RUnlock()
		assert.Equal(t, 3*time.Second, timing.duration, "two-second hydration plus one-second operate(), excluding the five-second lock wait")
		assert.Equal(t, time.Now(), timing.lastEnd)
		assert.Equal(t, 10*time.Second, controller.completionRequeueDelay(wf.UID, time.Now()), "hydration increases the gap to the requeue ceiling")
		attrs := attribute.NewSet()
		operation, err := testExporter.GetFloat64HistogramData(ctx, telemetry.InstrumentOperationDurationSeconds.Name(), &attrs)
		require.NoError(t, err)
		assert.Equal(t, uint64(1), operation.Count)
		assert.InDelta(t, 1.0, operation.Sum, 0.000001, "operation_duration_seconds still measures only operate()")
		repo.AssertExpectations(t)
	})
}
