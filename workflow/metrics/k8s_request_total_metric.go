package metrics

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v3/util/k8s"
)

var K8sRequestTotalMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: argoNamespace,
		Subsystem: workflowsSubsystem,
		Name:      "k8s_request_total",
		Help:      "Number of kubernetes requests executed. https://argo-workflows.readthedocs.io/en/latest/metrics/#argo_workflows_k8s_request_total",
	},
	[]string{"kind", "verb", "status_code"},
)

type metricsRoundTripper struct {
	roundTripper http.RoundTripper
}

func (m metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	x, err := m.roundTripper.RoundTrip(r)
	if x != nil {
		verb, kind := k8s.ParseRequest(r)
		K8sRequestTotalMetric.WithLabelValues(kind, verb, strconv.Itoa(x.StatusCode)).Inc()
	}
	return x, err
}

func AddMetricsTransportWrapper(config *rest.Config) *rest.Config {
	wrap := config.WrapTransport
	config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if wrap != nil {
			rt = wrap(rt)
		}
		return &metricsRoundTripper{roundTripper: rt}
	}
	return config
}
