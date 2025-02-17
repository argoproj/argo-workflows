package metrics

import "github.com/prometheus/client_golang/prometheus"

var PodMissingMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: argoNamespace,
		Name:      "pod_missing",
		Help:      "Incidents of pod missing. https://argo-workflows.readthedocs.io/en/release-3.5/metrics/#argo_pod_missing",
	},
	[]string{"recently_started", "node_phase"},
)
