package gccontroller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	testingclock "k8s.io/utils/clock/testing"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	fakewfclientset "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned/fake"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
	"github.com/argoproj/argo-workflows/v4/workflow/metrics"
	"github.com/argoproj/argo-workflows/v4/workflow/util"
)

func newTTLController(t *testing.T) *Controller {
	t.Helper()
	clock := testingclock.NewFakeClock(time.Now())
	wfclientset := fakewfclientset.NewClientset()
	wfInformer := cache.NewSharedIndexInformer(nil, nil, 0, nil)
	ctx := logging.TestContext(t.Context())
	gcMetrics, err := metrics.New(ctx, telemetry.TestScopeName, telemetry.TestScopeName, &telemetry.MetricsConfig{}, metrics.Callbacks{})
	require.NoError(t, err)
	return &Controller{
		wfclientset: wfclientset,
		wfInformer:  wfInformer,
		clock:       clock,
		workqueue:   workqueue.NewTypedDelayingQueueWithConfig[string](workqueue.TypedDelayingQueueConfig[string]{}),
		metrics:     gcMetrics,
		log:         logging.RequireLoggerFromContext(ctx),
	}
}

func enqueueWF(ctx context.Context, controller *Controller, un *unstructured.Unstructured) {
	controller.enqueueWF(ctx, un)
	time.Sleep(100*time.Millisecond + time.Second)
}

func TestEnqueueWF(t *testing.T) {
	var err error
	var un *unstructured.Unstructured

	controller := newTTLController(t)
	ctx := logging.TestContext(t.Context())

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow("@testdata/completed-workflow.yaml")
	un, err = util.ToUnstructured(wf)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())
}

func TestTTLStrategySucceeded(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController(t)
	ctx := logging.TestContext(t.Context())

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow("@testdata/succeeded-workflow.yaml")
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow("@testdata/succeeded-workflow.yaml")
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 1, controller.workqueue.Len())

	wf2 := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref-ttl-in-template.yaml")
	wf2.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf2)
	require.NoError(t, err)

	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf2, metav1.CreateOptions{})
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	controller.processNextWorkItem(ctx)
	assert.Equal(t, 1, controller.workqueue.Len())

	wf3 := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref-ttl-in-workflow.yaml")
	wf3.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf3)
	require.NoError(t, err)

	_, err = controller.wfclientset.ArgoprojV1alpha1().Workflows("default").Create(ctx, wf3, metav1.CreateOptions{})
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	controller.processNextWorkItem(ctx)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var ten int32 = 10

	controller := newTTLController(t)
	ctx := logging.TestContext(t.Context())

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 1, controller.workqueue.Len())
}

func TestNoTTLStrategyFailed(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	controller := newTTLController(t)
	ctx := logging.TestContext(t.Context())

	// Veirfy we do not enqueue if not completed
	wf := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	un, err = util.ToUnstructured(wf)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())

	wf1 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	un, err = util.ToUnstructured(wf1)
	require.NoError(t, err)
	enqueueWF(ctx, controller, un)
	assert.Equal(t, 0, controller.workqueue.Len())
}

func TestTTLStrategyFromUnstructured(t *testing.T) {
	var err error
	var un *unstructured.Unstructured
	var five int32 = 5

	controller3 := newTTLController(t)
	ctx := logging.TestContext(t.Context())
	wf3 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	ttlstrategy3 := wfv1.TTLStrategy{SecondsAfterSuccess: &five}
	wf3.Spec.TTLStrategy = &ttlstrategy3
	wf3.Status.FinishedAt = metav1.Time{Time: controller3.clock.Now().Add(-6 * time.Second)}
	un, err = util.ToUnstructured(wf3)
	t.Log(wf3.Spec.TTLStrategy)
	require.NoError(t, err)
	enqueueWF(ctx, controller3, un)
	assert.Equal(t, 0, controller3.workqueue.Len())
}

func TestTTLlExpired(t *testing.T) {
	controller := newTTLController(t)
	var ten int32 = 10

	wf := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	assert.True(t, wf.Status.Failed())
	now := controller.clock.Now()
	assert.True(t, now.After(wf.Status.FinishedAt.Add(time.Second*time.Duration(*wf.Spec.TTLStrategy.SecondsAfterFailure))))
	assert.True(t, wf.Status.Failed() && wf.Spec.TTLStrategy.SecondsAfterFailure != nil)
	expiresIn, ok := controller.expiresIn(wf)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf1 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf1.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf1.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf1)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf2 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf2.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterFailure: &ten}
	wf2.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf2)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf3 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf3.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompletion: &ten}
	wf3.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf3)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf4 := wfv1.MustUnmarshalWorkflow("@testdata/failed-workflow.yaml")
	wf4.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterCompletion: &ten}
	wf4.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf4)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)

	wf5 := wfv1.MustUnmarshalWorkflow("@testdata/succeeded-workflow.yaml")
	wf5.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf5.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-5 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf5)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, int(expiresIn), 0)

	wf6 := wfv1.MustUnmarshalWorkflow("@testdata/succeeded-workflow.yaml")
	wf6.Spec.TTLStrategy = &wfv1.TTLStrategy{SecondsAfterSuccess: &ten}
	wf6.Status.FinishedAt = metav1.Time{Time: controller.clock.Now().Add(-11 * time.Second)}
	expiresIn, ok = controller.expiresIn(wf6)
	assert.True(t, ok)
	assert.LessOrEqual(t, int(expiresIn), 0)
}

func TestGetTTLStrategy(t *testing.T) {
	var ten int32 = 10

	t.Run("TTLFromWorkflow", func(t *testing.T) {
		wf := wfv1.MustUnmarshalWorkflow("@testdata/succeeded-workflow.yaml")
		wf.Spec.TTLStrategy = &wfv1.TTLStrategy{
			SecondsAfterCompletion: &ten,
		}
		ttl := wf.GetTTLStrategy()
		assert.NotNil(t, ttl)
		assert.Equal(t, ten, *ttl.SecondsAfterCompletion)
	})

	t.Run("TTLInWfwithWorkflowTemplate", func(t *testing.T) {
		wf1 := wfv1.MustUnmarshalWorkflow("@testdata/workflow-template-ref-ttl-in-workflow.yaml")
		ttl := wf1.GetTTLStrategy()
		assert.NotNil(t, ttl)
		assert.Equal(t, ten, *ttl.SecondsAfterCompletion)

		wf1.Spec.TTLStrategy = nil
		wf1.Status.StoredWorkflowSpec.TTLStrategy = nil
		ttl = wf1.GetTTLStrategy()
		assert.Nil(t, ttl)
	})
}
