package metrics

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/env"

	tlsutils "github.com/argoproj/argo-workflows/v3/util/tls"
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
		metricsRegistry.MustRegister(collectors.NewGoCollector())
	} else if m.telemetryConfig.Enabled {
		// If the telemetry server is different -- and it's enabled -- run each on its own instance
		telemetryRegistry := prometheus.NewRegistry()
		telemetryRegistry.MustRegister(collectors.NewGoCollector())
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

	if config.Secure {
		tlsMinVersion, err := env.GetInt("TLS_MIN_VERSION", tls.VersionTLS12)
		if err != nil {
			panic(err)
		}
		log.Infof("Generating Self Signed TLS Certificates for Telemetry Servers")
		tlsConfig, err := tlsutils.GenerateX509KeyPairTLSConfig(uint16(tlsMinVersion))
		if err != nil {
			panic(err)
		}
		srv.TLSConfig = tlsConfig
		go func() {
			log.Infof("Starting prometheus metrics server at localhost:%v%s", config.Port, config.Path)
			if err := srv.ListenAndServeTLS("", ""); err != nil {
				panic(err)
			}
		}()
	} else {
		go func() {
			log.Infof("Starting prometheus metrics server at localhost:%v%s", config.Port, config.Path)
			if err := srv.ListenAndServe(); err != nil {
				panic(err)
			}
		}()
	}

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
	K8sRequestTotalMetric.Describe(ch)
	PodMissingMetric.Describe(ch)
	WorkflowConditionMetric.Describe(ch)
}

func (m *Metrics) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range m.allMetrics() {
		ch <- metric
	}
	m.logMetric.Collect(ch)
	K8sRequestTotalMetric.Collect(ch)
	PodMissingMetric.Collect(ch)
	WorkflowConditionMetric.Collect(ch)
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
