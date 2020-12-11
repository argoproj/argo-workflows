package metrics

import (
	"strconv"

	"github.com/argoproj/pkg/kubeclientmetrics"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
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
	log.WithFields(log.Fields{"kind": r.Kind, "namespace": r.Namespace, "name": r.Name, "verb": r.Verb}).Debug("IncKubernetesRequest")
	K8sRequestsTotal.WithLabelValues(r.Kind, string(r.Verb), strconv.Itoa(r.StatusCode)).Inc()
	return nil
}
