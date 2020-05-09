package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
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

type Metrics struct {
	metricsConfig   ServerConfig
	telemetryConfig ServerConfig

	workflowsProcessed prometheus.Counter
	workflowsByPhase   map[v1alpha1.NodePhase]prometheus.Gauge
	customMetrics      map[string]common.Metric
}

var _ prometheus.Collector = Metrics{}

func New(metricsConfig, telemetryConfig ServerConfig) Metrics {
	metrics := Metrics{
		metricsConfig:      metricsConfig,
		telemetryConfig:    telemetryConfig,
		workflowsProcessed: newCounter("workflows_processed", "Number of workflow updates processed", nil),
		workflowsByPhase:   getWorkflowPhaseGauges(),
		customMetrics:      make(map[string]common.Metric),
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
		allMetrics = append(allMetrics, metric.Metric)
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

func (m Metrics) GetCustomMetric(key string) common.Metric {
	// It's okay to return nil metrics in this function
	return m.customMetrics[key]
}

func (m Metrics) UpsertCustomMetric(key string, metric common.Metric) {
	m.customMetrics[key] = metric
}
