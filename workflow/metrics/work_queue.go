package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// Act as a metrics provider for a workqueues
var _ workqueue.MetricsProvider = &Metrics{}

// NOTE: This file implements external Kubernetes workqueue interfaces
// (workqueue.MetricsProvider, CounterMetric, GaugeMetric, HistogramMetric) which
// have fixed method signatures that cannot be modified to accept attribute parameters.
// It uses method values (closures) to capture queue names and call type-safe helpers
// internally, avoiding direct use of low-level instrument API.

type workersBusyRateLimiterWorkQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	addWorkersBusy func(ctx context.Context, val int64)
	// Evil storage of context for compatibility with legacy interface to workqueue
	//nolint:containedctx
	ctx context.Context
}

func addWorkQueueMetrics(_ context.Context, m *Metrics) error {
	instruments := []telemetry.BuiltinInstrument{
		telemetry.InstrumentWorkersBusyCount,
		telemetry.InstrumentQueueDepthGauge,
		telemetry.InstrumentQueueAddsCount,
		telemetry.InstrumentQueueLatency,
		telemetry.InstrumentQueueDuration,
		telemetry.InstrumentQueueRetries,
		telemetry.InstrumentQueueUnfinishedWork,
		telemetry.InstrumentQueueLongestRunning,
	}
	for _, inst := range instruments {
		if err := m.CreateBuiltinInstrument(inst); err != nil {
			return err
		}
	}

	// Setup observable gauge callbacks
	for _, inst := range []telemetry.BuiltinInstrument{
		telemetry.InstrumentQueueUnfinishedWork,
		telemetry.InstrumentQueueLongestRunning,
	} {
		ud := queueUserdata{gauge: m.GetInstrument(inst.Name())}
		ud.gauge.SetUserdata(&ud)
		if err := ud.gauge.RegisterCallback(m.Metrics, ud.update); err != nil {
			return err
		}
	}
	return nil
}

func (m *Metrics) RateLimiterWithBusyWorkers(ctx context.Context, workQueue workqueue.TypedRateLimiter[string], name string) workqueue.TypedRateLimitingInterface[string] {
	queue := workersBusyRateLimiterWorkQueue{
		TypedRateLimitingInterface: workqueue.NewTypedRateLimitingQueueWithConfig(workQueue, workqueue.TypedRateLimitingQueueConfig[string]{Name: name}),
		addWorkersBusy:             func(ctx context.Context, val int64) { m.AddWorkersBusyCount(ctx, val, name) },
		ctx:                        ctx,
	}
	queue.addWorkersBusy(ctx, 0) // Initialize worker count
	return queue
}

func (w workersBusyRateLimiterWorkQueue) Get() (string, bool) {
	item, shutdown := w.TypedRateLimitingInterface.Get()
	w.addWorkersBusy(w.ctx, 1)
	return item, shutdown
}

func (w workersBusyRateLimiterWorkQueue) Done(item string) {
	w.TypedRateLimitingInterface.Done(item)
	w.addWorkersBusy(w.ctx, -1)
}

// Shim between kubernetes queue interface and otel
type queueMetric struct {
	inc             func()
	dec             func()
	observe         func(val float64)
	set             func(val float64)
	value           *float64
	observeCallback func(ctx context.Context, o metric.Observer)
}

type queueUserdata struct {
	gauge   *telemetry.Instrument
	metrics []queueMetric
}

func (q queueMetric) Inc() {
	if q.inc != nil {
		q.inc()
	}
}

func (q queueMetric) Dec() {
	if q.dec != nil {
		q.dec()
	}
}

func (q queueMetric) Observe(val float64) {
	if q.observe != nil {
		q.observe(val)
	}
}

// Observable gauge stores in the shim
func (q queueMetric) Set(val float64) {
	if q.set != nil {
		q.set(val)
	}
}

func (m *Metrics) getQueueUserdata(ctx context.Context, i *telemetry.Instrument) *queueUserdata {
	switch val := i.GetUserdata().(type) {
	case *queueUserdata:
		return val
	default:
		m.fallbackLogger.WithField("metric", i.GetName()).Error(ctx, "internal error: unexpected userdata on queue metric")
		return &queueUserdata{}
	}
}

func (q *queueUserdata) update(ctx context.Context, o metric.Observer) error {
	for _, metric := range q.metrics {
		if metric.observeCallback != nil {
			metric.observeCallback(ctx, o)
		}
	}
	return nil
}

func (m *Metrics) NewDepthMetric(name string) workqueue.GaugeMetric {
	return queueMetric{
		inc: func() { m.AddQueueDepthGauge(context.Background(), 1, name) },
		dec: func() { m.AddQueueDepthGauge(context.Background(), -1, name) },
	}
}

func (m *Metrics) NewAddsMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		inc: func() { m.AddQueueAddsCount(context.Background(), 1, name) },
		dec: func() { m.AddQueueAddsCount(context.Background(), -1, name) },
	}
}

func (m *Metrics) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		observe: func(val float64) { m.RecordQueueLatency(context.Background(), val, name) },
	}
}

func (m *Metrics) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		observe: func(val float64) { m.RecordQueueDuration(context.Background(), val, name) },
	}
}

func (m *Metrics) NewRetriesMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		inc: func() { m.AddQueueRetries(context.Background(), 1, name) },
		dec: func() { m.AddQueueRetries(context.Background(), -1, name) },
	}
}

func (m *Metrics) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return m.newObservableGaugeMetric(name, telemetry.InstrumentQueueUnfinishedWork, m.ObserveQueueUnfinishedWork)
}

func (m *Metrics) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return m.newObservableGaugeMetric(name, telemetry.InstrumentQueueLongestRunning, m.ObserveQueueLongestRunning)
}

func (m *Metrics) newObservableGaugeMetric(name string, inst telemetry.BuiltinInstrument, observe func(context.Context, metric.Observer, float64, string)) queueMetric {
	valuePtr := ptr.To(float64(0.0))
	metric := queueMetric{
		value:           valuePtr,
		set:             func(val float64) { *valuePtr = val },
		observeCallback: func(ctx context.Context, o metric.Observer) { observe(ctx, o, *valuePtr, name) },
	}
	ud := m.getQueueUserdata(context.Background(), m.GetInstrument(inst.Name()))
	ud.metrics = append(ud.metrics, metric)
	return metric
}
