package metrics

import (
	"context"

	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/metric"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/pointer"
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
	busyGauge  *instrument
	// Evil storage of context for compatibility with legacy interface to workqueue
	ctx context.Context
}

func addWorkQueueMetrics(_ context.Context, m *Metrics) error {
	err := m.createInstrument(int64UpDownCounter,
		nameWorkersBusy,
		"Number of workers currently busy",
		"{worker}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(int64UpDownCounter,
		nameWorkersQueueDepth,
		"Depth of the queue",
		"{item}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(int64Counter,
		nameWorkersQueueAdds,
		"Adds to the queue",
		"{item}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(float64Histogram,
		nameWorkersQueueLatency,
		"Time objects spend waiting in the queue",
		"s",
		withDefaultBuckets([]float64{1.0, 5.0, 20.0, 60.0, 180.0}),
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(float64Histogram,
		nameWorkersQueueDuration,
		"Time objects spend being processed from the queue",
		"s",
		withDefaultBuckets([]float64{0.1, 0.2, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0, 60.0, 180.0}),
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(int64Counter,
		nameWorkersRetries,
		"Retries in the queues",
		"{item}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	err = m.createInstrument(float64ObservableGauge,
		nameWorkersUnfinishedWork,
		"Unfinished work time",
		"s",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	unfinishedCallback := queueUserdata{
		gauge: m.allInstruments[nameWorkersUnfinishedWork],
	}
	m.allInstruments[nameWorkersUnfinishedWork].userdata = &unfinishedCallback
	err = m.allInstruments[nameWorkersUnfinishedWork].registerCallback(m, unfinishedCallback.update)
	if err != nil {
		return err
	}

	err = m.createInstrument(float64ObservableGauge,
		nameWorkersLongestRunning,
		"Longest running worker",
		"s",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	longestRunningCallback := queueUserdata{
		gauge: m.allInstruments[nameWorkersLongestRunning],
	}
	m.allInstruments[nameWorkersLongestRunning].userdata = &longestRunningCallback
	err = m.allInstruments[nameWorkersLongestRunning].registerCallback(m, longestRunningCallback.update)
	if err != nil {
		return err
	}
	return nil
}

func (m *Metrics) RateLimiterWithBusyWorkers(ctx context.Context, workQueue workqueue.RateLimiter, queueName string) workqueue.RateLimitingInterface {
	queue := workersBusyRateLimiterWorkQueue{
		RateLimitingInterface: workqueue.NewNamedRateLimitingQueue(workQueue, queueName),
		workerType:            queueName,
		busyGauge:             m.allInstruments[nameWorkersBusy],
		ctx:                   ctx,
	}
	queue.newWorker(ctx)
	return queue
}

func (w *workersBusyRateLimiterWorkQueue) attributes() instAttribs {
	return instAttribs{{name: labelWorkerType, value: w.workerType}}
}

func (w *workersBusyRateLimiterWorkQueue) newWorker(ctx context.Context) {
	w.busyGauge.addInt(ctx, 0, w.attributes())
}

func (w *workersBusyRateLimiterWorkQueue) workerBusy(ctx context.Context) {
	w.busyGauge.addInt(ctx, 1, w.attributes())
}

func (w *workersBusyRateLimiterWorkQueue) workerFree(ctx context.Context) {
	w.busyGauge.addInt(ctx, -1, w.attributes())
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
	inst  *instrument
	value *float64
}

type queueUserdata struct {
	gauge   *instrument
	metrics []queueMetric
}

func (q *queueMetric) attributes() instAttribs {
	return instAttribs{{name: labelQueueName, value: q.name}}
}

func (q queueMetric) Inc() {
	q.inst.addInt(q.ctx, 1, q.attributes())
}

func (q queueMetric) Dec() {
	q.inst.addInt(q.ctx, -1, q.attributes())
}

func (q queueMetric) Observe(val float64) {
	q.inst.record(q.ctx, val, q.attributes())
}

// Observable gauge stores in the shim
func (q queueMetric) Set(val float64) {
	*(q.value) = val
}

func (i *instrument) queueUserdata() *queueUserdata {
	switch val := i.userdata.(type) {
	case *queueUserdata:
		return val
	default:
		log.Errorf("internal error: unexpected userdata on queue metric %s", i.name)
		return &queueUserdata{}
	}
}

func (q *queueUserdata) update(_ context.Context, o metric.Observer) error {
	for _, metric := range q.metrics {
		q.gauge.observeFloat(o, *metric.value, metric.attributes())
	}
	return nil
}

func (m *Metrics) NewDepthMetric(name string) workqueue.GaugeMetric {
	return queueMetric{
		ctx:  m.ctx,
		name: name,
		inst: m.allInstruments[nameWorkersQueueDepth],
	}
}

func (m *Metrics) NewAddsMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		ctx:  m.ctx,
		name: name,
		inst: m.allInstruments[nameWorkersQueueAdds],
	}
}

func (m *Metrics) NewLatencyMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		ctx:  m.ctx,
		name: name,
		inst: m.allInstruments[nameWorkersQueueLatency],
	}
}

func (m *Metrics) NewWorkDurationMetric(name string) workqueue.HistogramMetric {
	return queueMetric{
		ctx:  m.ctx,
		name: name,
		inst: m.allInstruments[nameWorkersQueueDuration],
	}
}

func (m *Metrics) NewRetriesMetric(name string) workqueue.CounterMetric {
	return queueMetric{
		ctx:  m.ctx,
		name: name,
		inst: m.allInstruments[nameWorkersRetries],
	}
}

func (m *Metrics) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		ctx:   m.ctx,
		name:  name,
		inst:  m.allInstruments[nameWorkersUnfinishedWork],
		value: pointer.Float64(0.0),
	}
	ud := metric.inst.queueUserdata()
	ud.metrics = append(ud.metrics, metric)
	return metric
}

func (m *Metrics) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	metric := queueMetric{
		ctx:   m.ctx,
		name:  name,
		inst:  m.allInstruments[nameWorkersLongestRunning],
		value: pointer.Float64(0.0),
	}
	ud := metric.inst.queueUserdata()
	ud.metrics = append(ud.metrics, metric)
	return metric
}
