package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
)

const (
	nameWorkersBusy           = `workers_busy_count`
	nameWorkersQueueDepth     = `queue_depth_gauge`
	nameWorkersQueueAdds      = `queue_adds_count`
	nameWorkersQueueLatency   = `queue_latency`
	nameWorkersQueueDuration  = `queue_duration`
	nameWorkersRetries        = `queue_retries`
	nameWorkersUnfinishedWork = `queue_unfinished_work`
	nameWorkersLongestRunning = `queue_longest_running`
)

// Act as a metrics provider for a workqueues
var _ workqueue.MetricsProvider = &Metrics{}

type workersBusyRateLimiterWorkQueue struct {
	workqueue.RateLimitingInterface
	workerType string
	busyGauge  *telemetry.Instrument
	// Evil storage of context for compatibility with legacy interface to workqueue
	ctx context.Context
}

func addWorkQueueMetrics(_ context.Context, m *Metrics) error {
	err := m.CreateInstrument(telemetry.Int64UpDownCounter,
		nameWorkersBusy,
		"Number of workers currently busy",
		"{worker}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Int64UpDownCounter,
		nameWorkersQueueDepth,
		"Depth of the queue",
		"{item}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Int64Counter,
		nameWorkersQueueAdds,
		"Adds to the queue",
		"{item}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Float64Histogram,
		nameWorkersQueueLatency,
		"Time objects spend waiting in the queue",
		"s",
		telemetry.WithDefaultBuckets([]float64{1.0, 5.0, 20.0, 60.0, 180.0}),
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Float64Histogram,
		nameWorkersQueueDuration,
		"Time objects spend being processed from the queue",
		"s",
		telemetry.WithDefaultBuckets([]float64{0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0, 60.0, 180.0}),
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Int64Counter,
		nameWorkersRetries,
		"Retries in the queues",
		"{item}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.CreateInstrument(telemetry.Float64ObservableGauge,
		nameWorkersUnfinishedWork,
		"Unfinished work time",
		"s",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	unfinishedCallback := queueUserdata{
		gauge: m.GetInstrument(nameWorkersUnfinishedWork),
	}
	unfinishedCallback.gauge.SetUserdata(&unfinishedCallback)
	err = unfinishedCallback.gauge.RegisterCallback(m.Metrics, unfinishedCallback.update)
	if err != nil {
		return err
	}

	err = m.CreateInstrument(telemetry.Float64ObservableGauge,
		nameWorkersLongestRunning,
		"Longest running worker",
		"s",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	longestRunningCallback := queueUserdata{
		gauge: m.GetInstrument(nameWorkersLongestRunning),
	}
	longestRunningCallback.gauge.SetUserdata(&longestRunningCallback)
	err = longestRunningCallback.gauge.RegisterCallback(m.Metrics, longestRunningCallback.update)
	if err != nil {
		return err
	}
	return nil
}

func (m *Metrics) RateLimiterWithBusyWorkers(ctx context.Context, workQueue workqueue.RateLimiter, queueName string) workqueue.RateLimitingInterface {
	queue := workersBusyRateLimiterWorkQueue{
		RateLimitingInterface: workqueue.NewNamedRateLimitingQueue(workQueue, queueName),
		workerType:            queueName,
		busyGauge:             m.GetInstrument(nameWorkersBusy),
		ctx:                   ctx,
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

func (w workersBusyRateLimiterWorkQueue) Get() (interface{}, bool) {
	item, shutdown := w.RateLimitingInterface.Get()
	w.workerBusy(w.ctx)
	return item, shutdown
}

func (w workersBusyRateLimiterWorkQueue) Done(item interface{}) {
	w.RateLimitingInterface.Done(item)
	w.workerFree(w.ctx)
}

// Shim between kubernetes queue interface and otel
type queueMetric struct {
	ctx   context.Context
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
	q.inst.AddInt(q.ctx, 1, q.attributes())
}

func (q queueMetric) Dec() {
	q.inst.AddInt(q.ctx, -1, q.attributes())
}

func (q queueMetric) Observe(val float64) {
	q.inst.Record(q.ctx, val, q.attributes())
}

// Observable gauge stores in the shim
func (q queueMetric) Set(val float64) {
	*(q.value) = val
}

func getQueueUserdata(i *telemetry.Instrument) *queueUserdata {
	switch val := i.GetUserdata().(type) {
	case *queueUserdata:
		return val
	default:
		log.Errorf("internal error: unexpected userdata on queue metric %s", i.GetName())
		return &queueUserdata{}
	}
}

func (q *queueUserdata) update(_ context.Context, o metric.Observer) error {
	for _, metric := range q.metrics {
		q.gauge.ObserveFloat(o, *metric.value, metric.attributes())
	}
	return nil
}

func (m *Metrics) NewDepthMetric(name string) workqueue.GaugeMetric {
	return queueMetric{
		ctx:  m.Ctx,
		name: name,
		inst: m.GetInstrument(nameWorkersQueueDepth),
	}
}

func (m *Metrics) NewAddsMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		ctx:  m.Ctx,
		name: name,
		inst: m.GetInstrument(nameWorkersQueueAdds),
	}
}

func (m *Metrics) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		ctx:  m.Ctx,
		name: name,
		inst: m.GetInstrument(nameWorkersQueueLatency),
	}
}

func (m *Metrics) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		ctx:  m.Ctx,
		name: name,
		inst: m.GetInstrument(nameWorkersQueueDuration),
	}
}

func (m *Metrics) NewRetriesMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		ctx:  m.Ctx,
		name: name,
		inst: m.GetInstrument(nameWorkersRetries),
	}
}

func (m *Metrics) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		ctx:   m.Ctx,
		name:  name,
		inst:  m.GetInstrument(nameWorkersUnfinishedWork),
		value: ptr.To(float64(0.0)),
	}
	ud := getQueueUserdata(metric.inst)
	ud.metrics = append(ud.metrics, metric)
	return metric
}

func (m *Metrics) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		ctx:   m.Ctx,
		name:  name,
		inst:  m.GetInstrument(nameWorkersLongestRunning),
		value: ptr.To(float64(0.0)),
	}
	ud := getQueueUserdata(metric.inst)
	ud.metrics = append(ud.metrics, metric)
	return metric
}
