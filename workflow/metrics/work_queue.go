package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

// Act as a metrics provider for a workqueues
var _ workqueue.MetricsProvider = &Metrics{}

type workersBusyRateLimiterWorkQueue struct {
	workqueue.TypedRateLimitingInterface[string]
	workerType string
	busyGauge  *telemetry.Instrument
	// Evil storage of context for compatibility with legacy interface to workqueue
	// nolint:containedctx
	ctx context.Context
}

func addWorkQueueMetrics(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentWorkersBusyCount)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueDepthGauge)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueAddsCount)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueLatency)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueDuration)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueRetries)
	if err != nil {
		return err
	}
	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueUnfinishedWork)
	if err != nil {
		return err
	}
	unfinishedCallback := queueUserdata{
		gauge: m.GetInstrument(telemetry.InstrumentQueueUnfinishedWork.Name()),
	}
	unfinishedCallback.gauge.SetUserdata(&unfinishedCallback)
	err = unfinishedCallback.gauge.RegisterCallback(m.Metrics, unfinishedCallback.update)
	if err != nil {
		return err
	}

	err = m.CreateBuiltinInstrument(telemetry.InstrumentQueueLongestRunning)
	if err != nil {
		return err
	}
	longestRunningCallback := queueUserdata{
		gauge: m.GetInstrument(telemetry.InstrumentQueueLongestRunning.Name()),
	}
	longestRunningCallback.gauge.SetUserdata(&longestRunningCallback)
	err = longestRunningCallback.gauge.RegisterCallback(m.Metrics, longestRunningCallback.update)
	if err != nil {
		return err
	}
	return nil
}

func (m *Metrics) RateLimiterWithBusyWorkers(ctx context.Context, workQueue workqueue.TypedRateLimiter[string], queueName string) workqueue.TypedRateLimitingInterface[string] {
	queue := workersBusyRateLimiterWorkQueue{
		TypedRateLimitingInterface: workqueue.NewTypedRateLimitingQueueWithConfig(workQueue, workqueue.TypedRateLimitingQueueConfig[string]{Name: queueName}),
		workerType:                 queueName,
		busyGauge:                  m.GetInstrument(telemetry.InstrumentWorkersBusyCount.Name()),
		ctx:                        ctx,
	}
	queue.newWorker(ctx)
	return queue
}

func (w *workersBusyRateLimiterWorkQueue) attributes() telemetry.InstAttribs {
	return telemetry.InstAttribs{{Name: telemetry.AttribWorkerType, Value: w.workerType}}
}

func (w *workersBusyRateLimiterWorkQueue) newWorker(ctx context.Context) {
	w.busyGauge.AddInt(ctx, 0, w.attributes())
}

func (w *workersBusyRateLimiterWorkQueue) workerBusy(ctx context.Context) {
	w.busyGauge.AddInt(ctx, 1, w.attributes())
}

func (w *workersBusyRateLimiterWorkQueue) workerFree(ctx context.Context) {
	w.busyGauge.AddInt(ctx, -1, w.attributes())
}

func (w workersBusyRateLimiterWorkQueue) Get() (string, bool) {
	item, shutdown := w.TypedRateLimitingInterface.Get()
	w.workerBusy(w.ctx)
	return item, shutdown
}

func (w workersBusyRateLimiterWorkQueue) Done(item string) {
	w.TypedRateLimitingInterface.Done(item)
	w.workerFree(w.ctx)
}

// Shim between kubernetes queue interface and otel
type queueMetric struct {
	name  string
	inst  *telemetry.Instrument
	value *float64
}

type queueUserdata struct {
	gauge   *telemetry.Instrument
	metrics []queueMetric
}

func (q *queueMetric) attributes() telemetry.InstAttribs {
	return telemetry.InstAttribs{{Name: telemetry.AttribQueueName, Value: q.name}}
}

func (q queueMetric) Inc() {
	q.inst.AddInt(context.Background(), 1, q.attributes())
}

func (q queueMetric) Dec() {
	q.inst.AddInt(context.Background(), -1, q.attributes())
}

func (q queueMetric) Observe(val float64) {
	q.inst.Record(context.Background(), val, q.attributes())
}

// Observable gauge stores in the shim
func (q queueMetric) Set(val float64) {
	*(q.value) = val
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
		q.gauge.ObserveFloat(ctx, o, *metric.value, metric.attributes())
	}
	return nil
}

func (m *Metrics) NewDepthMetric(name string) workqueue.GaugeMetric {
	return queueMetric{
		name: name,
		inst: m.GetInstrument(telemetry.InstrumentQueueDepthGauge.Name()),
	}
}

func (m *Metrics) NewAddsMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		name: name,
		inst: m.GetInstrument(telemetry.InstrumentQueueAddsCount.Name()),
	}
}

func (m *Metrics) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		name: name,
		inst: m.GetInstrument(telemetry.InstrumentQueueLatency.Name()),
	}
}

func (m *Metrics) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		name: name,
		inst: m.GetInstrument(telemetry.InstrumentQueueDuration.Name()),
	}
}

func (m *Metrics) NewRetriesMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		name: name,
		inst: m.GetInstrument(telemetry.InstrumentQueueRetries.Name()),
	}
}

func (m *Metrics) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		name:  name,
		inst:  m.GetInstrument(telemetry.InstrumentQueueUnfinishedWork.Name()),
		value: ptr.To(float64(0.0)),
	}
	ud := m.getQueueUserdata(context.Background(), metric.inst)
	ud.metrics = append(ud.metrics, metric)
	return metric
}

func (m *Metrics) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		name:  name,
		inst:  m.GetInstrument(telemetry.InstrumentQueueLongestRunning.Name()),
		value: ptr.To(float64(0.0)),
	}
	ud := m.getQueueUserdata(context.Background(), metric.inst)
	ud.metrics = append(ud.metrics, metric)
	return metric
}
