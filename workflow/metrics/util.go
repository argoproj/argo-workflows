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
	literal, computed, err := computeMetricValue(metricSpec.GetMetricValue(), wf)
	if err != nil {
		return nil, err
	}

	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	if computed != nil {
		if metric == nil {
			return prometheus.NewGaugeFunc(gaugeOpts, computed), nil
		}
		// When using computed metrics, there is no need to update them once they are created
		return metric, nil
	}

	if metric == nil {
		// This gauge has not been used before, create it
		gauge := prometheus.NewGauge(gaugeOpts)
		gauge.Set(literal)
		return gauge, nil
	}
	// This gauge exists, simply update it
	gauge := metric.(prometheus.Gauge)
	gauge.Set(literal)
	return gauge, nil
}

func computeMetricValue(value wfv1.MetricValue, wf *wfv1.Workflow) (float64, func() float64, error) {
	if value.Literal != "" {
		val, err := strconv.ParseFloat(value.Literal, 64)
		if err != nil {
			return 0.0, nil, err
		}
		return val, nil, nil
	}

	if value.Computed != "" {
		switch value.Computed {
		case wfv1.ComputedValueWorkflowDuration:
			if wf.Status.Completed() {
				// If the workflow is finished, return a literal value
				return wf.Status.FinishedAt.Time.Sub(wf.Status.StartedAt.Time).Seconds(), nil, nil
			}
			return 0.0, func() float64 {
				return time.Since(wf.Status.StartedAt.Time).Seconds()
			}, nil
		}
	}

	return 0.0, nil, fmt.Errorf("metric does not specify a value source")
}