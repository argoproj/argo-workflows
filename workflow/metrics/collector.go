package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

func NewMetricsRegistry(metrics Interface) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	return registry
}

// NewTelemetryRegistry creates a new prometheus registry that collects telemetry
func NewTelemetryRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{
		PidFn:        func() (int, error) { return os.Getpid(), nil },
		ReportErrors: true,
	}))
	registry.MustRegister(prometheus.NewGoCollector())
	return registry
}
