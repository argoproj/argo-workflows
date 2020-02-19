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

func ConstructOrUpdateMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, wf *wfv1.Workflow) (prometheus.Metric, error) {
	switch metricSpec.GetMetricType() {
	case wfv1.MetricTypeGauge:
		return constructOrUpdateGaugeMetric(metric, metricSpec, wf)
	default:
		return nil, fmt.Errorf("invalid metric spec")
	}
}

func constructOrUpdateGaugeMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, wf *wfv1.Workflow) (prometheus.Metric, error) {
	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	metricValue := metricSpec.GetMetricValue()

	if metricValue.Duration != nil {
		if metric == nil {
			return prometheus.NewGaugeFunc(gaugeOpts, func() float64 {
				if wf.Status.Completed() {
					return wf.Status.FinishedAt.Time.Sub(wf.Status.StartedAt.Time).Seconds()
				}
				return time.Since(wf.Status.StartedAt.Time).Seconds()
			}), nil
		}
		// When using duration metrics, there is no need to update them once they are created
		return metric, nil
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

func constructOrUpdateHistogramMetric(metric prometheus.Metric, metricSpec *wfv1.Metric, wf *wfv1.Workflow) (prometheus.Metric, error) {
	gaugeOpts := prometheus.HistogramOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
		Buckets:     metricSpec.Histogram.Bins,
	}

	metricValue := metricSpec.GetMetricValue()

	if metricValue.Duration != nil {
		var hist prometheus.Histogram
		if metric == nil {
			hist = prometheus.NewHistogram(gaugeOpts)
		} else {
			hist = metric.(prometheus.Histogram)
		}
		if wf.Status.Completed() {
			hist.Observe(wf.Status.FinishedAt.Time.Sub(wf.Status.StartedAt.Time).Seconds())
		}
		// When using duration metrics, there is no need to update them once they are created
		return metric, nil
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
