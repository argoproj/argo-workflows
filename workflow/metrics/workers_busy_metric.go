package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WorkersBusyMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Subsystem: workflowsSubsystem,
			Name:      "workers_busy",
			Help:      "Number of workers currently busy. https://argoproj.github.io/argo/metrics/#workers_busy",
		},
		[]string{"queue_name"},
	)
)
