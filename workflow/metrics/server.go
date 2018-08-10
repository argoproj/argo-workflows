package metrics

import (
	"context"
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

// RunServer starts a metrics server
func RunServer(ctx context.Context, config PrometheusConfig, registry *prometheus.Registry) {
	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: config.Port, Handler: mux}

	defer srv.Close()

	log.Infof("Starting prometheus metrics server at 0.0.0.0%s%s", config.Port, config.Path)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

	<-ctx.Done()
}
