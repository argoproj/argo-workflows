package metrics

import (
	"strconv"

	"github.com/argoproj/pkg/kubeclientmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	K8sRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: argoNamespace,
			Name:      "k8s_request_total",
			Help:      "Number of kubernetes requests executed",
		},
		[]string{"kind", "verb", "status_code"},
	)
)

func IncKubernetesRequest(r kubeclientmetrics.ResourceInfo) error {
	K8sRequestsTotal.WithLabelValues(r.Kind, string(r.Verb), strconv.Itoa(r.StatusCode)).Inc()
	return nil
}
