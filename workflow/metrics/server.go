package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// RunServer starts a metrics server
func (m Metrics) RunServer(stopCh <-chan struct{}) {
	mux := http.NewServeMux()
	mux.Handle(m.path, promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	srv := &http.Server{Addr: fmt.Sprintf(":%s", m.port), Handler: mux}

	defer func() {
		if cerr := srv.Close(); cerr != nil {
			log.Fatalf("Encountered an '%s' error when tried to close the metrics server running on '%s'", cerr, m.port)
		}
	}()

	go m.garbageCollector(stopCh)

	log.Infof("Starting prometheus metrics server at localhost:%s%s", m.port, m.path)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
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
	if m.ttl == 0 {
		return
	}

	ticker := time.NewTicker(m.ttl)
	defer ticker.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			for key, metric := range m.customMetrics {
				if time.Since(metric.LastUpdated) > m.ttl {
					delete(m.customMetrics, key)
				}
			}
		}
	}
}
