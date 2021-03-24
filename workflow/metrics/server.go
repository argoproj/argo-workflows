package metrics

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
)

// RunServer starts a metrics server
func (m *Metrics) RunServer(ctx context.Context) {
	defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

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
		go runServer(m.telemetryConfig, telemetryRegistry, ctx)
	}

	// Run the metrics server
	go runServer(m.metricsConfig, metricsRegistry, ctx)

	go m.garbageCollector(ctx)
}

func runServer(config ServerConfig, registry *prometheus.Registry, ctx context.Context) {
	var handlerOpts promhttp.HandlerOpts
	if config.IgnoreErrors {
		handlerOpts.ErrorHandling = promhttp.ContinueOnError
	}

	mux := http.NewServeMux()
	mux.Handle(config.Path, promhttp.HandlerFor(registry, handlerOpts))
	srv := &http.Server{Addr: fmt.Sprintf(":%v", config.Port), Handler: mux}

	go func() {
		log.Infof("Starting prometheus metrics server at localhost:%v%s", config.Port, config.Path)
		if err := srv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()

	// Waiting for stop signal
	<-ctx.Done()

	// Shutdown the server gracefully with a 1 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Infof("Unable to shutdown metrics server at localhost:%v%s", config.Port, config.Path)
	}
}

func (m *Metrics) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range m.allMetrics() {
		ch <- metric.Desc()
	}
	m.logMetric.Describe(ch)
	PodMissingMetric.Describe(ch)
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range m.allMetrics() {
		ch <- metric
	}
	m.logMetric.Collect(ch)
	PodMissingMetric.Collect(ch)
}

func (m *Metrics) garbageCollector(ctx context.Context) {
	if m.metricsConfig.TTL == 0 {
		return
	}

	ticker := time.NewTicker(m.metricsConfig.TTL)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			for key, metric := range m.customMetrics {
				if time.Since(metric.lastUpdated) > m.metricsConfig.TTL {
					delete(m.customMetrics, key)
				}
			}
		}
	}
}
