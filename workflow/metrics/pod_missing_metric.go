package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	PodMissingMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			// TODO Subsystem:   workflowsSubsystem,
			Name: "pod_missing",
			Help: "Incidents of pod missing. You should rarely see cases when the node is running except under high load.",
		},
		[]string{"recently_started", "node_phase"},
	)
)
