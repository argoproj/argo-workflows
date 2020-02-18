package metrics

import (
	"context"
	"fmt"
	"net/http"

	"github.com/argoproj/argo/workflow/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// RunServer starts a metrics server
func RunServer(ctx context.Context, config config.PrometheusConfig, registry *prometheus.Registry) {
	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: fmt.Sprintf(":%s", config.Port), Handler: mux}

	defer func() {
		if cerr := srv.Close(); cerr != nil {
			log.Fatalf("Encountered an '%s' error when tried to close the metrics server running on '%s'", cerr, config.Port)
		}
	}()

	log.Infof("Starting prometheus metrics server at 0.0.0.0:%s%s", config.Port, config.Path)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}

	<-ctx.Done()
}
