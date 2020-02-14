package metrics

import (
	"github.com/argoproj/argo/workflow/controller"
	"github.com/prometheus/client_golang/prometheus"
)

type customMetricsCollector struct {
	controller *controller.WorkflowController
}

// Describe implements the prometheus.Collector interface
func (cmc *customMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range cmc.controller.Metrics {
		ch <- metric.Desc()
	}
}

// Collect implements the prometheus.Collector interface
func (cmc *customMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	for _, metric := range cmc.controller.Metrics {
		ch <- metric
	}
}
