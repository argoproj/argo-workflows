package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WorkersBusyMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Name:      "worker_busy",
			Help:      "Number of workers currently busy",
		},
		[]string{"queue_name"},
	)
)
