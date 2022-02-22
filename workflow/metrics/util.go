package metrics

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

var (
	invalidMetricNameError = "metric name is invalid: names may only contain alphanumeric characters, '_', or ':'"
	invalidMetricLabelrror = "metric label '%s' is invalid: keys may only contain alphanumeric characters, '_', or ':'"
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
		labels := metricSpec.GetMetricLabels()
		if err := ValidateMetricLabels(labels); err != nil {
			return nil, err
		}
		metric = newCounter(metricSpec.Name, metricSpec.Help, labels)
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
		labels := metricSpec.GetMetricLabels()
		if err := ValidateMetricLabels(labels); err != nil {
			return nil, err
		}
		metric = newGauge(metricSpec.Name, metricSpec.Help, labels)
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
		labels := metricSpec.GetMetricLabels()
		if err := ValidateMetricLabels(labels); err != nil {
			return nil, err
		}
		metric = newHistogram(metricSpec.Name, metricSpec.Help, labels, metricSpec.Histogram.GetBuckets())
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
	getOptsByPhase := func(phase wfv1.NodePhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "count",
			Help:        "Number of Workflows currently accessible by the controller by status (refreshed every 15s)",
			ConstLabels: map[string]string{"status": string(phase)},
		}
	}
	return map[wfv1.NodePhase]prometheus.Gauge{
		wfv1.NodePending:   prometheus.NewGauge(getOptsByPhase(wfv1.NodePending)),
		wfv1.NodeRunning:   prometheus.NewGauge(getOptsByPhase(wfv1.NodeRunning)),
		wfv1.NodeSucceeded: prometheus.NewGauge(getOptsByPhase(wfv1.NodeSucceeded)),
		wfv1.NodeFailed:    prometheus.NewGauge(getOptsByPhase(wfv1.NodeFailed)),
		wfv1.NodeError:     prometheus.NewGauge(getOptsByPhase(wfv1.NodeError)),
	}
}

func getPodPhaseGauges() map[v1.PodPhase]prometheus.Gauge {
	getOptsByPhase := func(phase v1.PodPhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "pods_count",
			Help:        "Number of Pods from Workflows currently accessible by the controller by status (refreshed every 15s)",
			ConstLabels: map[string]string{"status": string(phase)},
		}
	}
	return map[v1.PodPhase]prometheus.Gauge{
		v1.PodPending: prometheus.NewGauge(getOptsByPhase(v1.PodPending)),
		v1.PodRunning: prometheus.NewGauge(getOptsByPhase(v1.PodRunning)),
		// v1.PodSucceeded: prometheus.NewGauge(getOptsByPhase(v1.PodSucceeded)),
		// v1.PodFailed:    prometheus.NewGauge(getOptsByPhase(v1.PodFailed)),
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
		ErrorCauseCronWorkflowSpecError:       prometheus.NewCounter(getOptsByPahse(ErrorCauseCronWorkflowSpecError)),
	}
}

func getWorkersBusy(name string) prometheus.Gauge {
	return prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        "workers_busy_count",
		Help:        "Number of workers currently busy",
		ConstLabels: map[string]string{"worker_type": name},
	})
}

func IsValidMetricName(name string) bool {
	return model.IsValidMetricName(model.LabelValue(name))
}

func ValidateMetricValues(metric *wfv1.Prometheus) error {
	if metric.Gauge != nil {
		if metric.Gauge.Value == "" {
			return errors.New("missing gauge.value")
		}
		if metric.Gauge.Realtime != nil && *metric.Gauge.Realtime {
			if strings.Contains(metric.Gauge.Value, "resourcesDuration.") {
				return errors.New("'resourcesDuration.*' metrics cannot be used in real-time")
			}
		}
	}
	if metric.Counter != nil && metric.Counter.Value == "" {
		return errors.New("missing counter.value")
	}
	if metric.Histogram != nil && metric.Histogram.Value == "" {
		return errors.New("missing histogram.value")
	}
	return nil
}

func ValidateMetricLabels(metrics map[string]string) error {
	for name := range metrics {
		if !IsValidMetricName(name) {
			return fmt.Errorf(invalidMetricLabelrror, name)
		}
	}
	return nil
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
