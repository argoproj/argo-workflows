package metrics

import (
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/cache"

	"github.com/argoproj/argo/workflow/util"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

type MetricsProvider interface {
	GetMetrics() []prometheus.Metric
	DeleteExpiredMetrics(ttl time.Duration)
}

func NewMetricsRegistry(metricsProvider MetricsProvider, informer cache.SharedIndexInformer, disableLegacyMetrics bool) *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(&customMetricsCollector{provider: metricsProvider})
	workflowLister := util.NewWorkflowLister(informer)
	registry.MustRegister(&controllerCollector{store: workflowLister})

	if !disableLegacyMetrics {
		registry.MustRegister(&legacyWorkflowCollector{store: workflowLister})
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
