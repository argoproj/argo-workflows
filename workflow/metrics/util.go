package metrics

import (
	"fmt"
	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

type RealTimeMetric struct {
	Func func () float64
}

func ConstructOrUpdateMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus, realTimeMetric RealTimeMetric) (prometheus.Metric, error) {
	switch metricSpec.GetMetricType() {
	case wfv1.MetricTypeGauge:
		return constructOrUpdateGaugeMetric(metric, metricSpec, realTimeMetric)
	case wfv1.MetricTypeHistogram:
		return constructOrUpdateHistogramMetric(metric, metricSpec)
	case wfv1.MetricTypeCounter:
		return constructOrUpdateCounterMetric(metric, metricSpec)
	default:
		return nil, fmt.Errorf("invalid metric spec")
	}
}

func constructOrUpdateCounterMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	counterOpts := prometheus.CounterOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	val, err := strconv.ParseFloat(metricSpec.Counter.Increment, 64)
	if err != nil {
		return nil, err
	}

	if metric == nil {
		metric = prometheus.NewCounter(counterOpts)
	}
	counter := metric.(prometheus.Counter)
	counter.Add(val)
	return counter, nil

}

func constructOrUpdateGaugeMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus, realTimeMetric RealTimeMetric) (prometheus.Metric, error) {
	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	if metricSpec.Gauge.RealTime != nil && *metricSpec.Gauge.RealTime {
		return prometheus.NewGaugeFunc(gaugeOpts, realTimeMetric.Func), nil
	}

	val, err := strconv.ParseFloat(metricSpec.Gauge.Value, 64)
	if err != nil {
		return nil, err
	}

	if metric == nil {
		// This gauge has not been used before, create it
		metric = prometheus.NewGauge(gaugeOpts)
	}
	// This gauge exists, simply update it
	gauge := metric.(prometheus.Gauge)
	gauge.Set(val)
	return gauge, nil
}

func constructOrUpdateHistogramMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	histOpts := prometheus.HistogramOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
		Buckets:     metricSpec.Histogram.Bins,
	}

	val, err := strconv.ParseFloat(metricSpec.Histogram.Value, 64)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		// This gauge has not been used before, create it
		metric = prometheus.NewHistogram(histOpts)
	}
	// This gauge exists, simply update it
	hist := metric.(prometheus.Histogram)
	hist.Observe(val)
	return hist, nil
}
