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
	Path string
	Port string
	TTL  time.Duration
}

type Metrics struct {
	registry     *prometheus.Registry
	serverConfig ServerConfig

	workflowsProcessed prometheus.Counter
	workflowsByPhase   map[v1alpha1.NodePhase]prometheus.Gauge
	customMetrics      map[string]common.Metric
}

var _ prometheus.Collector = Metrics{}

func New(config ServerConfig) Metrics {
	metrics := Metrics{
		serverConfig:       config,
		workflowsProcessed: newCounter("workflows_processed", "Number of workflow updates processed", nil),
		workflowsByPhase:   getWorkflowPhaseGauges(),
		customMetrics:      make(map[string]common.Metric),
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	registry.MustRegister(prometheus.NewGoCollector())
	metrics.registry = registry

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
