package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func write(metric prometheus.Metric) *dto.Metric {
	m := &dto.Metric{}
	err := metric.Write(m)
	if err != nil {
		panic(err)
	}
	return m
}

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

	require.True(t, a.SameServerAs(b))

	b.Enabled = false
	a.Enabled = false
	require.False(t, a.SameServerAs(b))

	b.Enabled = true
	a.Enabled = true
	b.Port = 9091
	require.False(t, a.SameServerAs(b))

	b.Port = DefaultMetricsServerPort
	b.Path = "/telemetry"
	require.False(t, a.SameServerAs(b))

	b.Path = DefaultMetricsServerPath
	b.Secure = true
	require.False(t, a.SameServerAs(b))
}

func TestMetrics(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	// Default buckets: {5, 10, 15, 20, 25, 30}
	m.OperationCompleted(5)
	require.Equal(t, 1, int(*write(m.operationDurations).Histogram.Bucket[1].CumulativeCount))

	require.Nil(t, m.GetCustomMetric("does-not-exist"))

	err := m.UpsertCustomMetric("metric", "", newCounter("test", "test", nil), false)
	require.NoError(t, err)
	require.NotNil(t, m.GetCustomMetric("metric"))

	err = m.UpsertCustomMetric("metric2", "", newCounter("test", "new test", nil), false)
	require.Error(t, err)

	badMetric, err := constructOrUpdateGaugeMetric(nil, &v1alpha1.Prometheus{
		Name:   "count",
		Help:   "Number of Workflows currently accessible by the controller by status (refreshed every 15s)",
		Labels: []*v1alpha1.MetricLabel{{Key: "status", Value: "Running"}},
		Gauge: &v1alpha1.Gauge{
			Value: "1",
		},
	})
	require.NoError(t, err)
	err = m.UpsertCustomMetric("asdf", "", badMetric, false)
	require.Error(t, err)
}

func TestErrors(t *testing.T) {
	_, err := ConstructRealTimeGaugeMetric(&v1alpha1.Prometheus{Name: "invalid.name"}, func() float64 { return 0.0 })
	require.Error(t, err)

	_, err = ConstructRealTimeGaugeMetric(&v1alpha1.Prometheus{Name: "name", Labels: []*v1alpha1.MetricLabel{{Key: "invalid-key", Value: "value"}}}, func() float64 { return 0.0 })
	require.Error(t, err)
}

func TestMetricGC(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
		TTL:     1 * time.Second,
	}
	m := New(config, config)
	require.Empty(t, m.customMetrics)

	err := m.UpsertCustomMetric("metric", "", newCounter("test", "test", nil), false)
	require.NoError(t, err)
	require.Len(t, m.customMetrics, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.garbageCollector(ctx)

	// Ensure we get at least one TTL run
	timeoutTime := time.Now().Add(time.Second * 2)
	for time.Now().Before(timeoutTime) {
		// Break if we know our test will pass.
		if len(m.customMetrics) == 0 {
			break
		}
		// Sleep to prevent overloading test worker CPU.
		time.Sleep(100 * time.Millisecond)
	}

	require.Empty(t, m.customMetrics)
}

func TestRealtimeMetricGC(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
		TTL:     1 * time.Second,
	}
	m := New(config, config)
	require.Empty(t, m.customMetrics)

	err := m.UpsertCustomMetric("realtime_metric", "workflow-uid", newCounter("test", "test", nil), true)
	require.NoError(t, err)
	require.Len(t, m.customMetrics, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.garbageCollector(ctx)

	// simulate workflow is still running.
	timeoutTime := time.Now().Add(time.Second * 2)
	// Ensure we get at least one TTL run
	for time.Now().Before(timeoutTime) {
		// Break if we know our test will pass.
		if len(m.customMetrics) == 0 {
			break
		}
		// Sleep to prevent overloading test worker CPU.
		time.Sleep(100 * time.Millisecond)
	}
	require.Len(t, m.customMetrics, 1)

	// simulate workflow is completed.
	m.StopRealtimeMetricsForKey("workflow-uid")
	timeoutTime = time.Now().Add(time.Second * 2)
	// Ensure we get at least one TTL run
	for time.Now().Before(timeoutTime) {
		// Break if we know our test will pass.
		if len(m.customMetrics) == 0 {
			break
		}
		// Sleep to prevent overloading test worker CPU.
		time.Sleep(100 * time.Millisecond)
	}
	require.Empty(t, m.customMetrics)
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
	defer wfQueue.ShutDown()

	require.NotNil(t, m.workqueueMetrics["workflow_queue-depth"])
	require.NotNil(t, m.workqueueMetrics["workflow_queue-adds"])
	require.NotNil(t, m.workqueueMetrics["workflow_queue-latency"])

	wfQueue.Add("hello")

	if require.NotNil(t, m.workqueueMetrics["workflow_queue-adds"]) {
		require.InEpsilon(t, 1.0, *write(m.workqueueMetrics["workflow_queue-adds"]).Counter.Value, 0.001)
	}
}

func TestRealTimeMetricDeletion(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
		TTL:     1 * time.Second,
	}
	m := New(config, config)

	rtMetric, err := ConstructRealTimeGaugeMetric(&v1alpha1.Prometheus{Name: "name", Help: "hello"}, func() float64 { return 0.0 })
	require.NoError(t, err)

	err = m.UpsertCustomMetric("metrickey", "123", rtMetric, true)
	require.NoError(t, err)
	require.NotEmpty(t, m.workflows["123"])
	require.Len(t, m.customMetrics, 1)

	m.DeleteRealtimeMetricsForKey("123")
	require.Empty(t, m.workflows["123"])
	require.Empty(t, m.customMetrics)

	metric, err := ConstructOrUpdateMetric(nil, &v1alpha1.Prometheus{Name: "name", Help: "hello", Gauge: &v1alpha1.Gauge{Value: "1"}})
	require.NoError(t, err)

	err = m.UpsertCustomMetric("metrickey", "456", metric, false)
	require.NoError(t, err)
	require.Empty(t, m.workflows["456"])
	require.Len(t, m.customMetrics, 1)

	m.DeleteRealtimeMetricsForKey("456")
	require.Empty(t, m.workflows["456"])
	require.Len(t, m.customMetrics, 1)
}
