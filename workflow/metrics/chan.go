package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ChanMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Name:      "channel_length",
			Help:      "Length of channel",
		},
		[]string{"chan_name"},
	)
)
