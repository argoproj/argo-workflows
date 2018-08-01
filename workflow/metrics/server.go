package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// PrometheusConfig defines a config for a metrics server
type PrometheusConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Path    string `json:"path,omitempty"`
	Port    string `json:"port,omitempty"`
}

// Server starts a metrics server
func Server(config PrometheusConfig, registry *prometheus.Registry) {
	if config.Enabled {
		mux := http.NewServeMux()
		mux.Handle(config.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		log.Infof("Starting prometheus metrics server at 0.0.0.0%s%s", config.Port, config.Path)
		http.ListenAndServe(config.Port, mux)
	}
}
