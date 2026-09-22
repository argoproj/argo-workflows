package metrics

import (
	"errors"
	"fmt"
	"strings"

	"github.com/prometheus/common/model"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

var (
	invalidMetricNameError  = "metric name is invalid: names may only contain alphanumeric characters or '_'"
	invalidMetricLabelError = "metric label '%s' is invalid: keys may only contain alphanumeric characters or '_'"
)

func IsValidMetricName(name string) bool {
	// Use promtheus's metric name checker, despite perhaps not using prometheus
	return model.LegacyValidation.IsValidMetricName(string(model.LabelValue(name))) && !strings.Contains(name, `:`)
}

// IsReservedMetricName reports whether name belongs to a controller metric.
// Controller and custom metrics share an instrument namespace, so they cannot
// safely use the same name.
func IsReservedMetricName(name string) bool {
	return telemetry.IsBuiltinInstrumentName(name)
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
			return fmt.Errorf(invalidMetricLabelError, name)
		}
	}
	return nil
}
