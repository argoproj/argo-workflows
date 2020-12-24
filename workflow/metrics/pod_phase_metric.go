package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	PodPhaseMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Subsystem: workflowsSubsystem,
			Name:      "pod_count",
			Help:      "Number of pods. https://argoproj.github.io/argo/fields/#pod_count",
		},
		[]string{"phase"},
	)
)
