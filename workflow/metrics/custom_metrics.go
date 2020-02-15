package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type customMetricsCollector struct {
	provider MetricsProvider
}

// Describe implements the prometheus.Collector interface
func (cmc *customMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range cmc.provider.GetMetrics() {
		ch <- metric().Desc()
	}
}

// Collect implements the prometheus.Collector interface
func (cmc *customMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range cmc.provider.GetMetrics() {
		ch <- metric()
	}
}
