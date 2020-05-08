package metrics

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
	"github.com/prometheus/client_golang/prometheus"
	"time"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

type Metrics struct {
	path     string
	port     string
	registry *prometheus.Registry
	ttl      time.Duration

	workflowsByPhase map[v1alpha1.NodePhase]prometheus.Gauge
	customMetrics    map[string]common.Metric
}

var _ prometheus.Collector = Metrics{}

func New(path, port string, ttl time.Duration) Metrics {
	metrics := Metrics{
		path:             path,
		port:             port,
		ttl:              ttl,
		workflowsByPhase: getWorkflowPhaseGauges(),
		customMetrics:    make(map[string]common.Metric),
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	registry.MustRegister(prometheus.NewGoCollector())
	metrics.registry = registry

	return metrics
}

func (m Metrics) allMetrics() []prometheus.Metric {
	var allMetrics []prometheus.Metric

	for _, metric := range m.workflowsByPhase {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.customMetrics {
		allMetrics = append(allMetrics, metric.Metric)
	}

	return allMetrics
}

func (m Metrics) AddWorkflowPhase(phase v1alpha1.NodePhase) {
	m.workflowsByPhase[phase].Inc()
}

func (m Metrics) DeleteWorkflowPhase(phase v1alpha1.NodePhase) {
	m.workflowsByPhase[phase].Dec()
}

func (m Metrics) GetCustomMetric(key string) common.Metric {
	// It's okay to return nil metrics in this function
	return m.customMetrics[key]
}

func (m Metrics) UpsertCustomMetric(key string, metric common.Metric) {
	m.customMetrics[key] = metric
}
