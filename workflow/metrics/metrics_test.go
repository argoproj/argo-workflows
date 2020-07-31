package metrics

import (
	"context"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func TestServerConfig_SameServerAs(t *testing.T) {
	a := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	b := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}

	assert.True(t, a.SameServerAs(b))

	b.Enabled = false
	a.Enabled = false
	assert.False(t, a.SameServerAs(b))

	b.Enabled = true
	a.Enabled = true
	b.Port = "9091"
	assert.False(t, a.SameServerAs(b))

	b.Port = DefaultMetricsServerPort
	b.Path = "/telemetry"
	assert.False(t, a.SameServerAs(b))
}

func TestMetrics(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	m.WorkflowAdded("wf", v1alpha1.NodeRunning)
	var metric dto.Metric
	err := m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(1), *metric.Gauge.Value)
	}

	// Test that we don't double add
	m.WorkflowAdded("wf", v1alpha1.NodeRunning)
	err = m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(1), *metric.Gauge.Value)
	}

	m.WorkflowUpdated("wf", v1alpha1.NodeRunning, v1alpha1.NodeSucceeded)
	err = m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(0), *metric.Gauge.Value)
	}
	err = m.workflowsByPhase[v1alpha1.NodeSucceeded].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(1), *metric.Gauge.Value)
	}

	m.WorkflowDeleted("wf", v1alpha1.NodeSucceeded)
	err = m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(0), *metric.Gauge.Value)
	}

	// Test that we don't double delete
	m.WorkflowDeleted("wf", v1alpha1.NodeSucceeded)
	err = m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(0), *metric.Gauge.Value)
	}

	// Test that we don't update workflows that we're not tracking
	m.WorkflowUpdated("does-not-exist", v1alpha1.NodeRunning, v1alpha1.NodeSucceeded)
	err = m.workflowsByPhase[v1alpha1.NodeRunning].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(0), *metric.Gauge.Value)
	}
	err = m.workflowsByPhase[v1alpha1.NodeSucceeded].Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, float64(0), *metric.Gauge.Value)
	}

	m.OperationCompleted(0.05)
	err = m.operationDurations.Write(&metric)
	if assert.NoError(t, err) {
		assert.Equal(t, uint64(1), *metric.Histogram.Bucket[0].CumulativeCount)
	}

	assert.Nil(t, m.GetCustomMetric("does-not-exist"))

	err = m.UpsertCustomMetric("metric", newCounter("test", "test", nil))
	if assert.NoError(t, err) {
		assert.NotNil(t, m.GetCustomMetric("metric"))
	}

	err = m.UpsertCustomMetric("metric2", newCounter("test", "new test", nil))
	assert.Error(t, err)

	badMetric, err := constructOrUpdateGaugeMetric(nil, &v1alpha1.Prometheus{
		Name:   "count",
		Help:   "Number of Workflows currently accessible by the controller by status",
		Labels: []*v1alpha1.MetricLabel{{Key: "status", Value: "Running"}},
		Gauge: &v1alpha1.Gauge{
			Value: "1",
		},
	})
	if assert.NoError(t, err) {
		err = m.UpsertCustomMetric("asdf", badMetric)
		assert.Error(t, err)
	}
}

func TestErrors(t *testing.T) {
	_, err := ConstructRealTimeGaugeMetric(&v1alpha1.Prometheus{Name: "invalid.name"}, func() float64 { return 0.0 })
	assert.Error(t, err)
}

func TestMetricGC(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
		TTL:     1 * time.Second,
	}
	m := New(config, config)
	assert.Len(t, m.customMetrics, 0)

	err := m.UpsertCustomMetric("metric", newCounter("test", "test", nil))
	if assert.NoError(t, err) {
		assert.Len(t, m.customMetrics, 1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.garbageCollector(ctx)

	// Ensure we get at least one TTL run
	time.Sleep(1*time.Second + time.Millisecond)

	assert.Len(t, m.customMetrics, 0)
}

func TestWorkflowQueueMetrics(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
		TTL:     1 * time.Second,
	}
	m := New(config, config)
	workqueue.SetProvider(m)
	wfQueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "workflow_queue")
	assert.NotNil(t, m.workqueueMetrics["workflow_queue-depth"])
	assert.NotNil(t, m.workqueueMetrics["workflow_queue-adds"])
	assert.NotNil(t, m.workqueueMetrics["workflow_queue-latency"])

	wfQueue.Add("hello")

	if assert.NotNil(t, m.workqueueMetrics["workflow_queue-adds"]) {
		var metric dto.Metric
		err := m.workqueueMetrics["workflow_queue-adds"].Write(&metric)
		if assert.NoError(t, err) {
			assert.Equal(t, 1.0, *metric.Counter.Value)
		}

	}
}
