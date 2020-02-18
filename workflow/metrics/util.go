package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func ConstructMetric(metric *v1alpha1.Metric, valueFn func() float64) MetricLoader {
	labelKeys, labelValues := metric.GetMetricLabels()

	var valueType prometheus.ValueType
	switch metric.GetMetricType() {
	case v1alpha1.MetricTypeGauge:
		valueType = prometheus.GaugeValue
	}

	metricDesc := prometheus.NewDesc(metric.Name, metric.Help, labelKeys, nil)
	return func() prometheus.Metric {
		value := valueFn()
		return prometheus.MustNewConstMetric(metricDesc, valueType, value, labelValues...)
	}
}
