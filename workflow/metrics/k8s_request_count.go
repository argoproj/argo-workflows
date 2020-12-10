package metrics

import (
	"strconv"

	"github.com/argoproj/pkg/kubeclientmetrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Custom events metric
	K8sRequestsCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: argoNamespace,
			Name:      "k8s_request_total",
			Help:      "Number of kubernetes requests executed during application reconciliation.",
		},
		[]string{"kind", "verb", "status_code"},
	)
)

func IncKubernetesRequest(resourceInfo kubeclientmetrics.ResourceInfo) error {
	if !resourceInfo.HasAllFields() {
		return nil
	}
	kind := resourceInfo.Kind
	statusCode := strconv.Itoa(resourceInfo.StatusCode)
	K8sRequestsCount.WithLabelValues(kind, string(resourceInfo.Verb), statusCode).Inc()
	return nil
}
