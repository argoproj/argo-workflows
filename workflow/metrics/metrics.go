package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	envutil "github.com/argoproj/argo-workflows/v3/util/env"
)

const (
	argoNamespace            = "argo"
	workflowsSubsystem       = "workflows"
	DefaultMetricsServerPort = 9090
	DefaultMetricsServerPath = "/metrics"
)

var (
	maxOperationTimeSeconds            = envutil.LookupEnvDurationOr("MAX_OPERATION_TIME", 30*time.Second).Seconds()
	operationDurationMetricBucketCount = envutil.LookupEnvIntOr("OPERATION_DURATION_METRIC_BUCKET_COUNT", 6)
)

type ServerConfig struct {
	Enabled      bool
	Path         string
	Port         int
	TTL          time.Duration
	IgnoreErrors bool
	Secure       bool
}

func (s ServerConfig) SameServerAs(other ServerConfig) bool {
	return s.Port == other.Port && s.Path == other.Path && s.Enabled && other.Enabled && s.Secure == other.Secure
}

type metric struct {
	metric      prometheus.Metric
	lastUpdated time.Time
}

type Metrics struct {
	// Ensures mutual exclusion in workflows map
	mutex           sync.RWMutex
	metricsConfig   ServerConfig
	telemetryConfig ServerConfig

	workflowsProcessed prometheus.Counter
	podsByPhase        map[corev1.PodPhase]prometheus.Gauge
	workflowsByPhase   map[v1alpha1.NodePhase]prometheus.Gauge
	workflows          map[string][]string
	operationDurations prometheus.Histogram
	errors             map[ErrorCause]prometheus.Counter
	customMetrics      map[string]metric
	workqueueMetrics   map[string]prometheus.Metric
	workersBusy        map[string]prometheus.Gauge

	// Used to quickly check if a metric desc is already used by the system
	defaultMetricDescs map[string]bool
	metricNameHelps    map[string]string
	logMetric          *prometheus.CounterVec
}

func (m *Metrics) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.WarnLevel, log.ErrorLevel}
}

func (m *Metrics) Fire(entry *log.Entry) error {
	m.logMetric.WithLabelValues(entry.Level.String()).Inc()
	return nil
}

var _ prometheus.Collector = &Metrics{}

func New(metricsConfig, telemetryConfig ServerConfig) *Metrics {
	bucketWidth := maxOperationTimeSeconds / float64(operationDurationMetricBucketCount)
	metrics := &Metrics{
		metricsConfig:      metricsConfig,
		telemetryConfig:    telemetryConfig,
		workflowsProcessed: newCounter("workflows_processed_count", "Number of workflow updates processed", nil),
		podsByPhase:        getPodPhaseGauges(),
		workflowsByPhase:   getWorkflowPhaseGauges(),
		workflows:          make(map[string][]string),
		operationDurations: newHistogram("operation_duration_seconds",
			"Histogram of durations of operations",
			nil,
			// We start the bucket with `bucketWidth` since lowest bucket has an upper bound of 'start'.
			prometheus.LinearBuckets(bucketWidth, bucketWidth, operationDurationMetricBucketCount)),
		errors:             getErrorCounters(),
		customMetrics:      make(map[string]metric),
		workqueueMetrics:   make(map[string]prometheus.Metric),
		workersBusy:        make(map[string]prometheus.Gauge),
		defaultMetricDescs: make(map[string]bool),
		metricNameHelps:    make(map[string]string),
		logMetric: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "log_messages",
			Help: "Total number of log messages.",
		}, []string{"level"}),
	}

	for _, metric := range metrics.allMetrics() {
		metrics.defaultMetricDescs[metric.Desc().String()] = true
	}

	for _, level := range metrics.Levels() {
		metrics.logMetric.WithLabelValues(level.String())
	}

	log.AddHook(metrics)

	return metrics
}

func (m *Metrics) allMetrics() []prometheus.Metric {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	allMetrics := []prometheus.Metric{
		m.workflowsProcessed,
		m.operationDurations,
	}
	for _, metric := range m.workflowsByPhase {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.podsByPhase {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.errors {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.workqueueMetrics {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.workersBusy {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.customMetrics {
		allMetrics = append(allMetrics, metric.metric)
	}
	return allMetrics
}

func (m *Metrics) StopRealtimeMetricsForKey(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.workflows[key]; !exists {
		return
	}

	realtimeMetrics := m.workflows[key]
	for _, metric := range realtimeMetrics {
		delete(m.customMetrics, metric)
	}

	delete(m.workflows, key)
}

func (m *Metrics) OperationCompleted(durationSeconds float64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.operationDurations.Observe(durationSeconds)
}

func (m *Metrics) GetCustomMetric(key string) prometheus.Metric {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// It's okay to return nil metrics in this function
	return m.customMetrics[key].metric
}

func (m *Metrics) UpsertCustomMetric(key string, ownerKey string, newMetric prometheus.Metric, realtime bool) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	metricDesc := newMetric.Desc().String()
	if _, inUse := m.defaultMetricDescs[metricDesc]; inUse {
		return fmt.Errorf("metric '%s' is already in use by the system, please use a different name", newMetric.Desc())
	}
	name, help := recoverMetricNameAndHelpFromDesc(metricDesc)
	if existingHelp, inUse := m.metricNameHelps[name]; inUse && help != existingHelp {
		return fmt.Errorf("metric '%s' has help string '%s' but should have '%s' (help strings must be identical for metrics of the same name)", name, help, existingHelp)
	} else {
		m.metricNameHelps[name] = help
	}
	m.customMetrics[key] = metric{metric: newMetric, lastUpdated: time.Now()}

	// If this is a realtime metric, track it
	if realtime {
		m.workflows[ownerKey] = append(m.workflows[ownerKey], key)
	}

	return nil
}

func (m *Metrics) SetWorkflowPhaseGauge(phase v1alpha1.NodePhase, num int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.workflowsByPhase[phase].Set(float64(num))
}

func (m *Metrics) SetPodPhaseGauge(phase corev1.PodPhase, num int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.podsByPhase[phase].Set(float64(num))
}

type ErrorCause string

const (
	ErrorCauseOperationPanic              ErrorCause = "OperationPanic"
	ErrorCauseCronWorkflowSubmissionError ErrorCause = "CronWorkflowSubmissionError"
	ErrorCauseCronWorkflowSpecError       ErrorCause = "CronWorkflowSpecError"
)

func (m *Metrics) OperationPanic() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.errors[ErrorCauseOperationPanic].Inc()
}

func (m *Metrics) CronWorkflowSubmissionError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.errors[ErrorCauseCronWorkflowSubmissionError].Inc()
}

func (m *Metrics) CronWorkflowSpecError() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.errors[ErrorCauseCronWorkflowSpecError].Inc()
}

// Act as a metrics provider for a workflow queue
var _ workqueue.MetricsProvider = &Metrics{}

func (m *Metrics) NewDepthMetric(name string) workqueue.GaugeMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s-depth", name)
	if _, ok := m.workqueueMetrics[key]; !ok {
		m.workqueueMetrics[key] = newGauge("queue_depth_count", "Depth of the queue", map[string]string{"queue_name": name})
	}
	return m.workqueueMetrics[key].(prometheus.Gauge)
}

func (m *Metrics) NewAddsMetric(name string) workqueue.CounterMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s-adds", name)
	if _, ok := m.workqueueMetrics[key]; !ok {
		m.workqueueMetrics[key] = newCounter("queue_adds_count", "Adds to the queue", map[string]string{"queue_name": name})
	}
	return m.workqueueMetrics[key].(prometheus.Counter)
}

func (m *Metrics) NewLatencyMetric(name string) workqueue.HistogramMetric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s-latency", name)
	if _, ok := m.workqueueMetrics[key]; !ok {
		m.workqueueMetrics[key] = newHistogram("queue_latency", "Time objects spend waiting in the queue", map[string]string{"queue_name": name}, []float64{1.0, 5.0, 20.0, 60.0, 180.0})
	}
	return m.workqueueMetrics[key].(prometheus.Histogram)
}

// These metrics are not relevant to be exposed
type noopMetric struct{}

func (noopMetric) Inc()            {}
func (noopMetric) Dec()            {}
func (noopMetric) Set(float64)     {}
func (noopMetric) Observe(float64) {}

func (m *Metrics) NewRetriesMetric(name string) workqueue.CounterMetric        { return noopMetric{} }
func (m *Metrics) NewWorkDurationMetric(name string) workqueue.HistogramMetric { return noopMetric{} }
func (m *Metrics) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return noopMetric{}
}

func (m *Metrics) NewLongestRunningProcessorSecondsMetric(name string) workqueue.SettableGaugeMetric {
	return noopMetric{}
}
