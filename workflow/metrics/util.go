package metrics

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var (
	invalidMetricNameError = "metric name is invalid: names may only contain alphanumeric characters, '_', or ':'"
	descRegex              = regexp.MustCompile(fmt.Sprintf(`Desc{fqName: "%s_%s_(.+?)", help: "(.+?)", constLabels: {`, argoNamespace, workflowsSubsystem))
)

type RealTimeMetric struct {
	Func func() float64
}

func ConstructOrUpdateMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	if !IsValidMetricName(metricSpec.Name) {
		return nil, fmt.Errorf(invalidMetricNameError)
	}

	switch metricSpec.GetMetricType() {
	case wfv1.MetricTypeGauge:
		return constructOrUpdateGaugeMetric(metric, metricSpec)
	case wfv1.MetricTypeHistogram:
		return constructOrUpdateHistogramMetric(metric, metricSpec)
	case wfv1.MetricTypeCounter:
		return constructOrUpdateCounterMetric(metric, metricSpec)
	default:
		return nil, fmt.Errorf("invalid metric spec")
	}
}

func ConstructRealTimeGaugeMetric(metricSpec *wfv1.Prometheus, valueFunc func() float64) (prometheus.Metric, error) {
	if !IsValidMetricName(metricSpec.Name) {
		return nil, fmt.Errorf(invalidMetricNameError)
	}

	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        metricSpec.Name,
		Help:        metricSpec.Help,
		ConstLabels: metricSpec.GetMetricLabels(),
	}

	return prometheus.NewGaugeFunc(gaugeOpts, valueFunc), nil
}

func constructOrUpdateCounterMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	if metric == nil {
		metric = newCounter(metricSpec.Name, metricSpec.Help, metricSpec.GetMetricLabels())
	}

	val, err := strconv.ParseFloat(metricSpec.Counter.Value, 64)
	if err != nil {
		return nil, err
	}

	counter := metric.(prometheus.Counter)
	counter.Add(val)
	return counter, nil
}

func constructOrUpdateGaugeMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	if metric == nil {
		metric = newGauge(metricSpec.Name, metricSpec.Help, metricSpec.GetMetricLabels())
	}

	val, err := strconv.ParseFloat(metricSpec.Gauge.Value, 64)
	if err != nil {
		return nil, err
	}

	gauge := metric.(prometheus.Gauge)
	gauge.Set(val)
	return gauge, nil
}

func constructOrUpdateHistogramMetric(metric prometheus.Metric, metricSpec *wfv1.Prometheus) (prometheus.Metric, error) {
	if metric == nil {
		metric = newHistogram(metricSpec.Name, metricSpec.Help, metricSpec.GetMetricLabels(), metricSpec.Histogram.GetBuckets())
	}

	val, err := strconv.ParseFloat(metricSpec.Histogram.Value, 64)
	if err != nil {
		return nil, err
	}

	hist := metric.(prometheus.Histogram)
	hist.Observe(val)
	return hist, nil
}

func newCounter(name, help string, labels map[string]string) prometheus.Counter {
	counterOpts := prometheus.CounterOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        name,
		Help:        help,
		ConstLabels: labels,
	}
	m := prometheus.NewCounter(counterOpts)
	mustBeRecoverable(name, help, m)
	return m
}

func newGauge(name, help string, labels map[string]string) prometheus.Gauge {
	gaugeOpts := prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        name,
		Help:        help,
		ConstLabels: labels,
	}
	m := prometheus.NewGauge(gaugeOpts)
	mustBeRecoverable(name, help, m)
	return m
}

func newHistogram(name, help string, labels map[string]string, buckets []float64) prometheus.Histogram {
	histOpts := prometheus.HistogramOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        name,
		Help:        help,
		ConstLabels: labels,
		Buckets:     buckets,
	}
	m := prometheus.NewHistogram(histOpts)
	mustBeRecoverable(name, help, m)
	return m
}

func getWorkflowPhaseGauges() map[wfv1.NodePhase]prometheus.Gauge {
	getOptsByPahse := func(phase wfv1.NodePhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "count",
			Help:        "Number of Workflows currently accessible by the controller by status",
			ConstLabels: map[string]string{"status": string(phase)},
		}
	}
	return map[wfv1.NodePhase]prometheus.Gauge{
		wfv1.NodePending:   prometheus.NewGauge(getOptsByPahse(wfv1.NodePending)),
		wfv1.NodeRunning:   prometheus.NewGauge(getOptsByPahse(wfv1.NodeRunning)),
		wfv1.NodeSucceeded: prometheus.NewGauge(getOptsByPahse(wfv1.NodeSucceeded)),
		wfv1.NodeSkipped:   prometheus.NewGauge(getOptsByPahse(wfv1.NodeSkipped)),
		wfv1.NodeFailed:    prometheus.NewGauge(getOptsByPahse(wfv1.NodeFailed)),
		wfv1.NodeError:     prometheus.NewGauge(getOptsByPahse(wfv1.NodeError)),
	}
}

func getErrorCounters() map[ErrorCause]prometheus.Counter {
	getOptsByPahse := func(phase ErrorCause) prometheus.CounterOpts {
		return prometheus.CounterOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "error_count",
			Help:        "Number of errors encountered by the controller by cause",
			ConstLabels: map[string]string{"cause": string(phase)},
		}
	}
	return map[ErrorCause]prometheus.Counter{
		ErrorCauseOperationPanic:              prometheus.NewCounter(getOptsByPahse(ErrorCauseOperationPanic)),
		ErrorCauseCronWorkflowSubmissionError: prometheus.NewCounter(getOptsByPahse(ErrorCauseCronWorkflowSubmissionError)),
	}
}

func IsValidMetricName(name string) bool {
	return model.IsValidMetricName(model.LabelValue(name))
}

func mustBeRecoverable(name, help string, metric prometheus.Metric) {
	recoveredName, recoveredHelp := recoverMetricNameAndHelpFromDesc(metric.Desc().String())
	if name != recoveredName {
		panic(fmt.Sprintf("unable to recover metric name from desc provided by prometheus: expected '%s' got '%s'", name, recoveredName))
	}
	if help != recoveredHelp {
		panic(fmt.Sprintf("unable to recover metric help from desc provided by prometheus: expected '%s' got '%s'", help, recoveredHelp))
	}
}

func recoverMetricNameAndHelpFromDesc(desc string) (string, string) {
	finds := descRegex.FindStringSubmatch(desc)
	if len(finds) != 3 {
		panic(fmt.Sprintf("malformed desc provided by prometheus: '%s' parsed to %v", desc, finds))
	}
	return finds[1], strings.ReplaceAll(finds[2], `\"`, `"`)
}
