package metrics

import (
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

func ConstructOrUpdateMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, emitter wfv1.MetricsEmitter) (prometheus.Metric, error) {
	switch metricSpec.GetMetricType() {
	case wfv1.MetricTypeGauge:
		return constructOrUpdateGaugeMetric(metric, metricSpec, emitter)
	case wfv1.MetricTypeHistogram:
		return constructOrUpdateHistogramMetric(metric, metricSpec, emitter)
	case wfv1.MetricTypeCounter:
		return constructOrUpdateCounterMetric(metric, metricSpec, emitter)
	default:
		return nil, fmt.Errorf("invalid metric spec")
	}
}

func constructOrUpdateCounterMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, emitter wfv1.MetricsEmitter) (prometheus.Metric, error) {

	prometheus.NewCounter()
}

func constructOrUpdateGaugeMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, emitter wfv1.MetricsEmitter) (prometheus.Metric, error) {
	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	metricValue := metricSpec.GetMetricValue()

	if metricValue.Duration != "" {
		switch metricValue.Duration {
		case wfv1.DurationTypeRealTime:
			// When using real time duration, update the function every time
			return prometheus.NewGaugeFunc(gaugeOpts, func() float64 {
				if emitter.Completed() {
					return emitter.FinishTime().Time.Sub(emitter.StartTime().Time).Seconds()
				}
				return time.Since(emitter.StartTime().Time).Seconds()
			}), nil
		case wfv1.DurationTypeOnCompletion:
			if metric == nil {
				metric = prometheus.NewGauge(gaugeOpts)
			}
			if emitter.Completed() {
				metric := metric.(prometheus.Gauge)
				metric.Set(emitter.FinishTime().Time.Sub(emitter.StartTime().Time).Seconds())
			}
			return metric, nil
		default:
			return nil, fmt.Errorf("unknown Duration value '%s' for metric '%s'", metricValue.Duration, metricSpec.Name)
		}
	}

	val, err := strconv.ParseFloat(metricValue.Literal, 64)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		// This gauge has not been used before, create it
		gauge := prometheus.NewGauge(gaugeOpts)
		gauge.Set(val)
		return gauge, nil
	}
	// This gauge exists, simply update it
	gauge := metric.(prometheus.Gauge)
	gauge.Set(val)
	return gauge, nil
}

func constructOrUpdateHistogramMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, emitter wfv1.MetricsEmitter) (prometheus.Metric, error) {
	histOpts := prometheus.HistogramOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
		Buckets:     metricSpec.Histogram.Bins,
	}

	metricValue := metricSpec.GetMetricValue()

	if metricValue.Duration != "" {
		switch metricValue.Duration {
		case wfv1.DurationTypeOnCompletion:
			var hist prometheus.Histogram
			if metric == nil {
				hist = prometheus.NewHistogram(histOpts)
			} else {
				hist = metric.(prometheus.Histogram)
			}
			if emitter.Completed() {
				hist.Observe(emitter.FinishTime().Time.Sub(emitter.StartTime().Time).Seconds())
			}
			// When using duration metrics, there is no need to update them once they are created
			return hist, nil
		case wfv1.DurationTypeRealTime:
			return nil, fmt.Errorf("unable to use real time duration with histograms")
		default:
			return nil, fmt.Errorf("unknown Duration value '%s' for metric '%s'", metricValue.Duration, metricSpec.Name)
		}
	}

	val, err := strconv.ParseFloat(metricValue.Literal, 64)
	if err != nil {
		return nil, err
	}
	if metric == nil {
		// This gauge has not been used before, create it
		hist := prometheus.NewHistogram(histOpts)
		hist.Observe(val)
		return hist, nil
	}
	// This gauge exists, simply update it
	hist := metric.(prometheus.Histogram)
	hist.Observe(val)
	return hist, nil
}
