package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

type ServerConfig struct {
	Enabled bool
	Path    string
	Port    string
	TTL     time.Duration
}

func (s ServerConfig) SameServerAs(other ServerConfig) bool {
	return s.Port == other.Port && s.Path == other.Path && s.Enabled && other.Enabled
}

type metric struct {
	metric      prometheus.Metric
	lastUpdated time.Time
}

type Metrics struct {
	metricsConfig   ServerConfig
	telemetryConfig ServerConfig

	workflowsProcessed prometheus.Counter
	workflowsByPhase   map[v1alpha1.NodePhase]prometheus.Gauge
	customMetrics      map[string]metric

	// Used to quickly check if a metric desc is already used by the system
	defaultMetricDescs map[string]bool
}

var _ prometheus.Collector = Metrics{}

func New(metricsConfig, telemetryConfig ServerConfig) Metrics {
	metrics := Metrics{
		metricsConfig:      metricsConfig,
		telemetryConfig:    telemetryConfig,
		workflowsProcessed: newCounter("workflows_processed", "Number of workflow updates processed", nil),
		workflowsByPhase:   getWorkflowPhaseGauges(),
		customMetrics:      make(map[string]metric),
		defaultMetricDescs: make(map[string]bool),
	}

	for _, metric := range metrics.allMetrics() {
		metrics.defaultMetricDescs[metric.Desc().String()] = true
	}

	return metrics
}

func (m Metrics) allMetrics() []prometheus.Metric {
	allMetrics := []prometheus.Metric{
		m.workflowsProcessed,
	}
	for _, metric := range m.workflowsByPhase {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.customMetrics {
		allMetrics = append(allMetrics, metric.metric)
	}

	return allMetrics
}

func (m Metrics) WorkflowAdded(phase v1alpha1.NodePhase) {
	if _, ok := m.workflowsByPhase[phase]; ok {
		m.workflowsByPhase[phase].Inc()
	}
}

func (m Metrics) WorkflowUpdated(fromPhase, toPhase v1alpha1.NodePhase) {
	m.WorkflowDeleted(fromPhase)
	m.WorkflowAdded(toPhase)
}

func (m Metrics) WorkflowDeleted(phase v1alpha1.NodePhase) {
	if _, ok := m.workflowsByPhase[phase]; ok {
		m.workflowsByPhase[phase].Dec()
	}
}

func (m Metrics) GetCustomMetric(key string) prometheus.Metric {
	// It's okay to return nil metrics in this function
	return m.customMetrics[key].metric
}

func (m Metrics) UpsertCustomMetric(key string, newMetric prometheus.Metric) error {
	if _, inUse := m.defaultMetricDescs[newMetric.Desc().String()]; inUse {
		return fmt.Errorf("metric '%s' is already in use by the system, please use a different name", newMetric.Desc())
	}
	m.customMetrics[key] = metric{metric: newMetric, lastUpdated: time.Now()}
	return nil
}
