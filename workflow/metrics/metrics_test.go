package metrics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func TestMetrics(t *testing.T) {
	m, te, err := CreateDefaultTestMetrics()
	if assert.NoError(t, err) {
		// Default buckets: {5, 10, 15, 20, 25, 30}
		m.OperationCompleted(m.ctx, 5)
		assert.NotNil(t, te)
		attribs := attribute.NewSet()
		val, err := te.GetFloat64HistogramData(nameOperationDuration, &attribs)
		assert.NoError(t, err)
		assert.Equal(t, []float64{5, 10, 15, 20, 25, 30}, val.Bounds)
		assert.Equal(t, []uint64{1, 0, 0, 0, 0, 0, 0}, val.BucketCounts)
	}
}

func TestErrors(t *testing.T) {
	m, _, err := CreateDefaultTestMetrics()

	assert.Nil(t, m.GetCustomMetric("does-not-exist"))

	assert.NoError(t, err)
	err = m.UpsertCustomMetric(m.ctx, &wfv1.Prometheus{
		Name: "invalid.name",
	}, "owner", func() float64 { return 0.0 })
	assert.Error(t, err)

	err = m.UpsertCustomMetric(m.ctx, &wfv1.Prometheus{
		Name: "name",
		Labels: []*wfv1.MetricLabel{{
			Key:   "invalid-key",
			Value: "value",
		}},
	}, "owner", func() float64 { return 0.0 })
	assert.Error(t, err)
}

func TestMetricGC(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
		TTL:     1 * time.Second,
	}
	m, _, err := createTestMetrics(&config, Callbacks{})
	if assert.NoError(t, err) {
		const key string = `metric`

		labels := []*wfv1.MetricLabel{
			{Key: "foo", Value: "bar"},
		}
		err = m.UpsertCustomMetric(m.ctx, &wfv1.Prometheus{
			Name:    key,
			Labels:  labels,
			Help:    "none",
			Counter: &wfv1.Counter{Value: "0.0"},
		}, "owner", nil)
		assert.NoError(t, err)
		baseCm := m.GetCustomMetric(key)
		assert.NotEqual(t, nil, baseCm)

		cm := baseCm.customUserdata()
		assert.Len(t, cm, 1)

		// Ensure we get at least one TTL run
		timeoutTime := time.Now().Add(time.Second * 2)
		for time.Now().Before(timeoutTime) {
			// Break if we know our test will pass.
			if len(cm) == 0 {
				break
			}
			// Sleep to prevent overloading test worker CPU.
			time.Sleep(100 * time.Millisecond)
		}

		assert.Len(t, cm, 0)
	}
}

func TestRealtimeMetricGC(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
		TTL:     1 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := New(ctx, TestScopeName, &config, Callbacks{})
	if assert.NoError(t, err) {
		//		assert.Len(t, m.customMetrics, 0)

		labels := []*wfv1.MetricLabel{
			{Key: "foo", Value: "bar"},
		}
		name := "realtime_metric"
		wfKey := "workflow-uid"
		err := m.UpsertCustomMetric(m.ctx, &wfv1.Prometheus{
			Name:   name,
			Labels: labels,
			Help:   "None",
			Gauge: &wfv1.Gauge{
				Realtime: pointer.Bool(true),
			}},
			wfKey,
			func() float64 { return 1.0 },
		)
		assert.NoError(t, err)
		assert.Len(t, m.realtimeWorkflows[wfKey], 1)

		go m.customMetricsGC(ctx, config.TTL)

		// simulate workflow is still running.
		// ensure we get at least one TTL run
		time.Sleep(time.Second * 2)
		assert.Len(t, m.realtimeWorkflows[wfKey], 1)

		// simulate workflow is completed.
		m.StopRealtimeMetricsForWfUID(wfKey)
		timeoutTime := time.Now().Add(time.Second * 2)
		// Ensure we get at least one TTL run
		for time.Now().Before(timeoutTime) {
			// Break if we know our test will pass.
			if len(m.realtimeWorkflows[wfKey]) == 0 {
				break
			}
			// Sleep to prevent overloading test worker CPU.
			time.Sleep(100 * time.Millisecond)
		}
		assert.Len(t, m.realtimeWorkflows[wfKey], 0)
	}
}

func TestWorkflowQueueMetrics(t *testing.T) {
	m, te, err := getSharedMetrics()
	if assert.NoError(t, err) {
		attribs := attribute.NewSet(attribute.String(labelQueueName, "workflow_queue"))
		wfQueue := m.RateLimiterWithBusyWorkers(m.ctx, workqueue.DefaultControllerRateLimiter(), "workflow_queue")
		defer wfQueue.ShutDown()

		assert.NotNil(t, m.allInstruments[nameWorkersQueueDepth])
		assert.NotNil(t, m.allInstruments[nameWorkersQueueLatency])

		wfQueue.Add("hello")

		if assert.NotNil(t, m.allInstruments[nameWorkersQueueAdds]) {
			val, err := te.GetInt64CounterValue(nameWorkersQueueAdds, &attribs)
			if assert.NoError(t, err) {
				assert.Equal(t, int64(1), val)
			}
		}
	}
}

func TestRealTimeMetricDeletion(t *testing.T) {
	config := Config{
		Enabled: true,
		Path:    defaultPrometheusServerPath,
		Port:    defaultPrometheusServerPort,
		TTL:     1 * time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m, err := New(ctx, TestScopeName, &config, Callbacks{})
	assert.NoError(t, err)

	// We've not yet fed a metric in for 123
	m.StopRealtimeMetricsForWfUID("123")
	assert.Len(t, m.realtimeWorkflows["123"], 0)

	const key string = `metric`

	labels := []*wfv1.MetricLabel{
		{Key: "foo", Value: "bar"},
	}
	err = m.UpsertCustomMetric(ctx, &wfv1.Prometheus{
		Name:   key,
		Labels: labels,
		Help:   "hello",
		Gauge: &wfv1.Gauge{
			Value:     "1.0",
			Realtime:  pointer.Bool(true),
			Operation: wfv1.GaugeOperationAdd,
		},
	}, "123", func() float64 { return 0.0 })
	assert.NoError(t, err)

	baseCm := m.GetCustomMetric(key)
	assert.NotEqual(t, nil, baseCm)

	m.StopRealtimeMetricsForWfUID("456")
	assert.Len(t, m.realtimeWorkflows["456"], 0)

	cm := baseCm.customUserdata()
	assert.Len(t, cm, 1)
	assert.Len(t, m.realtimeWorkflows["123"], 1)

	m.StopRealtimeMetricsForWfUID("123")
	assert.Len(t, m.realtimeWorkflows["123"], 0)
	assert.Len(t, cm, 0)

	err = m.UpsertCustomMetric(ctx, &wfv1.Prometheus{
		Name:   key,
		Labels: labels,
		Help:   "hello",
		Gauge: &wfv1.Gauge{
			Value:     "1.0",
			Realtime:  pointer.Bool(true),
			Operation: wfv1.GaugeOperationAdd,
		},
	}, "456", nil)
	assert.NoError(t, err)

	assert.Len(t, cm, 1)
	assert.Len(t, m.realtimeWorkflows["456"], 1)

	// TODO - this is a busted test
	m.StopRealtimeMetricsForWfUID("456")
	//	assert.Empty(t, m.workflows["123"])
	assert.Len(t, m.realtimeWorkflows["456"], 0)
	assert.Len(t, cm, 0)
	// metric, err := ConstructOrUpdateMetric(nil, &v1alpha1.Prometheus{Name: "name", Help: "hello", Gauge: &v1alpha1.Gauge{Value: "1"}})
	// assert.NoError(t, err)

	// err = m.UpsertCustomMetric("metrickey", "456", metric, false)
	// assert.NoError(t, err)
	// assert.Empty(t, m.workflows["456"])
	// assert.Len(t, m.customMetrics, 1)

	// m.StopRealtimeMetricsForKey("456")
	// assert.Empty(t, m.workflows["456"])
	// assert.Len(t, m.customMetrics, 1)
}
