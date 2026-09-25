package controller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestBasicMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/basic-metric.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Schedule first pod and mark completed
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	metricName := wf.Spec.Templates[0].Metrics.Prometheus[0].Name
	assert.True(t, controller.metrics.CustomMetricExists(metricName))
	attribs := attribute.NewSet(attribute.String("name", "random-int"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, metricName, &attribs)
	require.NoError(t, err)
}

func TestGaugeMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/gauge-metric.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Schedule first pod and mark completed
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("name", "random-int"))

	valAdd, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(10.0), valAdd, 0.001)

	valSub, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[1].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(-5.0), valSub, 0.001)

	valSet, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[2].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(50.0), valSet, 0.001)

	valDefault, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[3].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(15.0), valDefault, 0.001)
}

func TestCounterMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/counter-metric.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()

	// Schedule first pod and mark completed
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodFailed)

	// Process first metrics
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("name", "flakey"))

	valTotal, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valTotal, 0.001)

	valError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[1].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valError, 0.001)
}

func TestMetricEmissionSameOperationCreationAndFailure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/metric-emission-same-operation-creation-and-failure.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet()

	valError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valError, 0.001)
}

func TestRetryStrategyMetric(t *testing.T) {
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/retry-strategy-metric.yaml")
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx, wf)
	defer cancel()
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx)

	// Ensure no metrics have been emitted yet
	metricErrorDesc := wf.Spec.Templates[0].Metrics.Prometheus[0].GetKey()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))
	metricErrorDesc = wf.Spec.Templates[1].Metrics.Prometheus[0].GetKey()
	assert.Nil(t, controller.metrics.GetCustomMetric(metricErrorDesc))

	// Simulate pod succeeded
	podNode := woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"]
	podNode.Phase = v1alpha1.NodeSucceeded
	woc.wf.Status.Nodes["workflow-template-whalesay-9pk8f-1966833540"] = podNode
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx)

	attribs := attribute.NewSet()

	valWfError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[0].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1.0), valWfError, 0.001)

	valTplError, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1.0), valTplError, 0.001)
}

func TestDAGTmplMetrics(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/dag-tmpl-metrics.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	makePodsPhase(ctx, woc, apiv1.PodSucceeded)
	woc.operate(ctx)

	attribs := attribute.NewSet()
	tmpl := woc.wf.GetTemplateByName("random-int")
	assert.NotNil(t, tmpl)

	val, err := testExporter.GetFloat64HistogramData(ctx, tmpl.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InEpsilon(t, float64(5.0), val.Sum, 0.001)
	assert.Equal(t, uint64(1), val.Count)

	attribs = attribute.NewSet(attribute.String("name", "flakey"), attribute.String("status", "Failed"))
	tmpl = woc.wf.GetTemplateByName("flakey")
	assert.NotNil(t, tmpl)
	valErrCount, err := testExporter.GetFloat64CounterValue(ctx, tmpl.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), valErrCount, 0.001)
}

func TestRealtimeWorkflowMetric(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/realtime-workflow-metric.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("label", "foobar"), attribute.String("workflowName", "test-foobar"))
	value, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	// The realtime value is derived from time.Since(Status.StartedAt), and StartedAt
	// carries no monotonic reading, so this reads the wall clock. Two back-to-back
	// collections land in the same wall clock tick on platforms with a coarse timer
	// (~0.5ms on Windows) and report an identical duration, so wait out a tick before
	// asserting the gauge has advanced.
	time.Sleep(10 * time.Millisecond)
	value1, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	t.Logf("%v new %v old", value1, value)
	assert.Greater(t, value1, value)

	ctx = woc.markWorkflowSuccess(ctx)
	value2, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	value3, err := testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	// Duration should be same after workflow complete
	assert.InEpsilon(t, value2, value3, 0.001)
}

func TestRealtimeWorkflowMetricWithGlobalParameters(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/realtime-workflow-metric-with-global-parameters.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("label", "foobar"), attribute.String("workflowName", "test-foobar"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
}

func TestProcessedRetryNode(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/processed-retry-node.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)

	attribs := attribute.NewSet(attribute.String("work_unit", "metrics-eg::A"), attribute.String("workflow_result", "Succeeded"))
	value, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Templates[1].Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), value, 0.001)
}

func TestControllerRestartWithRunningWorkflow(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/suspend-wf-with-metrics.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)

	woc.operate(ctx)
	attribs := attribute.NewSet(attribute.String("name", "model_a"))
	_, err = testExporter.GetFloat64GaugeValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
}

func TestRuntimeMetrics(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()
	wfcset := controller.wfclientset.ArgoprojV1alpha1().Workflows("")
	wf := v1alpha1.MustUnmarshalWorkflow("@testdata/operator_metrics/runtime-wf-metrics.yaml")
	_, err := wfcset.Create(ctx, wf, metav1.CreateOptions{})
	require.NoError(t, err)
	woc := newWorkflowOperationCtx(ctx, wf, controller)
	woc.operate(ctx) // create step node

	makePodsPhase(ctx, woc, apiv1.PodSucceeded) // pod is successful - manually workflow is succeeded
	woc = newWorkflowOperationCtx(ctx, woc.wf, controller)
	woc.operate(ctx) // node status of previous context

	attribs := attribute.NewSet(attribute.String("playground_id_workflow_counter", "test"), attribute.String("status", "Succeeded"))
	value, err := testExporter.GetFloat64CounterValue(ctx, woc.wf.Spec.Metrics.Prometheus[0].Name, &attribs)
	require.NoError(t, err)
	assert.InDelta(t, float64(1), value, 0.001)
}
