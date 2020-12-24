package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PodCountMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Subsystem: workflowsSubsystem,
			Name:      "pod_count",
			Help:      "Number of pods. https://argoproj.github.io/argo/metrics/#pod_count",
		},
		[]string{"phase"},
	)
)
