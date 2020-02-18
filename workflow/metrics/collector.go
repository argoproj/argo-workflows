package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/workflow/util"
)

// TODO: What's the best place for these?
type MetricsProvider interface {
	GetMetrics() map[string]MetricLoader
}

type MetricLoader func() prometheus.Metric

func NewMetricsRegistry(metricsProvider MetricsProvider, informer cache.SharedIndexInformer, includeLegacyMetrics bool) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(&customMetricsCollector{provider: metricsProvider})

	if includeLegacyMetrics {
		workflowLister := util.NewWorkflowLister(informer)
		registry.MustRegister(&workflowCollector{store: workflowLister})
	}

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
