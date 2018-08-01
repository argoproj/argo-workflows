package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	addr       = ":8080"
	namespaces = []string{"default"}
)

// ServeMetrics registers workflow collector and starts a /metrics server.
func ServeMetrics() {
	kubeClient, err := createKubeClient()
	if err != nil {
		return
	}
	registry := prometheus.NewRegistry()
	registerWfCollector(registry, kubeClient, namespaces)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.ListenAndServe(addr, nil)
}
