package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	PodGCMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Name:      "pod_gc",
			Help:      "Number of API requests executed to GC pods",
		},
		[]string{"deletion_method"},
	)
)
