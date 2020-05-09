package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// RunServer starts a metrics server
func (m Metrics) RunServer(stopCh <-chan struct{}) {
	if !m.metricsConfig.Enabled {
		// If metrics aren't enabled, return
		return
	}

	metricsRegistry := prometheus.NewRegistry()
	metricsRegistry.MustRegister(m)

	if m.metricsConfig.SameServerAs(m.telemetryConfig) {
		// If the metrics and telemetry servers are the same, run both of them in the same instance
		metricsRegistry.MustRegister(prometheus.NewGoCollector())
	} else if m.telemetryConfig.Enabled {
		// If the telemetry server is different -- and it's enabled -- run each on its own instance
		telemetryRegistry := prometheus.NewRegistry()
		telemetryRegistry.MustRegister(prometheus.NewGoCollector())
		go runServer(m.telemetryConfig.Path, m.telemetryConfig.Port, telemetryRegistry, stopCh)
	}

	// Run the metrics server
	go runServer(m.metricsConfig.Path, m.metricsConfig.Port, metricsRegistry, stopCh)

	go m.garbageCollector(stopCh)
}

func runServer(path, port string, registry *prometheus.Registry, stopCh <-chan struct{}) {
	mux := http.NewServeMux()
	mux.Handle(path, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: fmt.Sprintf(":%s", port), Handler: mux}

	go func() {
		log.Infof("Starting prometheus metrics server at localhost:%s%s", port, path)
		if err := srv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// Waiting for stop signal
	<-stopCh

	// Shutdown the server gracefully with a 1 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Infof("Unable to shutdown metrics server at localhost:%s%s", port, path)
	}
}

func (m Metrics) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range m.allMetrics() {
		ch <- metric.Desc()
	}
}

func (m Metrics) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range m.allMetrics() {
		ch <- metric
	}
}

func (m Metrics) garbageCollector(stopCh <-chan struct{}) {
	if m.metricsConfig.TTL == 0 {
		return
	}

	ticker := time.NewTicker(m.metricsConfig.TTL)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			for key, metric := range m.customMetrics {
				if time.Since(metric.LastUpdated) > m.metricsConfig.TTL {
					delete(m.customMetrics, key)
				}
			}
		}
	}
}
