package metrics

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/rest"
)

var (
	K8sRequestTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: argoNamespace,
			Subsystem: workflowsSubsystem,
			Name:      "k8s_request_total",
			Help:      "Number of kubernetes requests executed. https://argoproj.github.io/argo/metrics/#argo_workflows_k8s_request_total",
		},
		[]string{"kind", "verb", "status_code"},
	)
)

type metricsRoundTripper struct {
	roundTripper http.RoundTripper
}

func (m metricsRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	x, err := m.roundTripper.RoundTrip(r)
	if x != nil {
		verb, kind := parseRequest(r)
		K8sRequestTotalMetric.WithLabelValues(kind, verb, strconv.Itoa(x.StatusCode)).Inc()
	}
	return x, err
}

func parseRequest(r *http.Request) (verb string, kind string) {
	i := strings.Index(r.URL.Path, "/v") + 1
	path := strings.Split(r.URL.Path[i:], "/")
	n := len(path)

	verb = map[string]string{
		http.MethodGet:    "List",
		http.MethodPost:   "Create",
		http.MethodDelete: "Delete",
		http.MethodPatch:  "Patch",
		http.MethodPut:    "Update",
	}[r.Method]

	if r.URL.Query().Get("watch") != "" {
		verb = "Watch"
	} else if verb == "List" && n%2 == 1 {
		verb = "Get"
	} else if verb == "Delete" && n%2 == 0 {
		verb = "DeleteCollection"
	}

	kind = "Unknown"
	switch verb {
	case "List", "Watch", "Create", "DeleteCollection":
		kind = path[n-1]
	case "Get", "Delete", "Patch", "Update":
		kind = path[n-2]
	}

	return verb, kind
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
