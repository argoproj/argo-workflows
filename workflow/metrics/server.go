package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

type ServerConfigs map[int]ServerConfig

func (c ServerConfigs) Add(port int, path string, registry *prometheus.Registry) {
	log.Infof("padding %s", path)
	_, ok := c[port]
	if ok {
		c[port][path] = registry
	} else {
		c[port] = ServerConfig{path: registry}
	}
}

type ServerConfig map[string]*prometheus.Registry

// RunServer starts a metrics server
func (c ServerConfig) RunServer(ctx context.Context, port int) {
	mux := http.NewServeMux()
	for path, registry := range c {
		mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	}
	srv := &http.Server{Addr: fmt.Sprintf(":%v", port), Handler: mux}

	defer func() {
		if cerr := srv.Close(); cerr != nil {
			log.Fatalf("Encountered an '%s' error when tried to close the metrics server running on '%v'", cerr, port)
		}
	}()

	log.Infof("Starting Prometheus server at 0.0.0.0:%v{%s}", port, strings.Join(c.paths(), ","))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("failed to start Prometheus server: %v", err)
	}

	<-ctx.Done()
}

func (c ServerConfig) paths() []string {
	var paths []string
	for path := range c {
		paths = append(paths, path)
	}
	return paths
}
