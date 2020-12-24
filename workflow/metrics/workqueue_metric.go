package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WorkersBusyMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			// TODO Subsystem:   workflowsSubsystem,
			Name: "worker_busy",
			Help: "Number of workers currently busy. https://argoproj.github.io/argo/fields/#worker_busy",
		},
		[]string{"queue_name"},
	)
)
