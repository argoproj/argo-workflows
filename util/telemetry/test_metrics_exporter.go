package telemetry

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v3/util/logging"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

// TestScopeName is the name that the metrics running under test will have
const TestScopeName string = "argo-workflows-test"

// TestExporter is an opentelemetry metrics exporter, purely for use within
// tests. It is not possible to query the values of an instrument via the otel
// SDK, so this exporter provides methods by which you can request
// metrics by name+attributes and therefore inspect whether they exist, and
// their values for the purposes of testing only.
// This is a public structure as it is used outside of this module also.
type TestMetricsExporter struct {
	metric.Reader
}

var _ metric.Reader = &TestMetricsExporter{}

func NewTestMetricsExporter() *TestMetricsExporter {
	reader := metric.NewManualReader()

	e := &TestMetricsExporter{
		Reader: reader,
	}
	return e
}

func (t *TestMetricsExporter) getOurMetrics() (*[]metricdata.Metrics, error) {
	metrics := metricdata.ResourceMetrics{}
	err := t.Collect(logging.WithLogger(context.TODO(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())), &metrics)
	if err != nil {
		return nil, err
	}
	for _, scope := range metrics.ScopeMetrics {
		if scope.Scope.Name == TestScopeName {
			return &scope.Metrics, nil
		}
	}
	return nil, fmt.Errorf("%s scope not found", TestScopeName)
}

func (t *TestMetricsExporter) getNamedMetric(name string) (*metricdata.Metrics, error) {
	mtcs, err := t.getOurMetrics()
	if err != nil {
		return nil, err
	}
	for _, metric := range *mtcs {
		if metric.Name == name {
			return &metric, nil
		}
	}
	return nil, fmt.Errorf("%s named metric not found in %v", name, mtcs)
}

func (t *TestMetricsExporter) getNamedInt64CounterData(name string, attribs *attribute.Set) (*metricdata.DataPoint[int64], error) {
	mtc, err := t.getNamedMetric(name)
	if err != nil {
		return nil, err
	}

	if counter, ok := mtc.Data.(metricdata.Sum[int64]); ok {
		for _, dataPoint := range counter.DataPoints {
			if dataPoint.Attributes.Equals(attribs) {
				return &dataPoint, nil
			}
		}
		return nil, fmt.Errorf("%s named counter[int64] with attribs %v not found in %v", name, attribs, mtc)
	}
	return nil, fmt.Errorf("%s type counter[int64] not found in %v", name, mtc)
}

func (t *TestMetricsExporter) getNamedFloat64GaugeData(name string, attribs *attribute.Set) (*metricdata.DataPoint[float64], error) {
	mtc, err := t.getNamedMetric(name)
	if err != nil {
		return nil, err
	}

	if gauge, ok := mtc.Data.(metricdata.Gauge[float64]); ok {
		for _, dataPoint := range gauge.DataPoints {
			if dataPoint.Attributes.Equals(attribs) {
				return &dataPoint, nil
			}
		}
		return nil, fmt.Errorf("%s named gauge[float64] with attribs %v not found in %v", name, attribs, mtc)
	}
	return nil, fmt.Errorf("%s type gauge[float64] not found in %v", name, mtc)
}

func (t *TestMetricsExporter) getNamedInt64GaugeData(name string, attribs *attribute.Set) (*metricdata.DataPoint[int64], error) {
	mtc, err := t.getNamedMetric(name)
	if err != nil {
		return nil, err
	}

	gauge, ok := mtc.Data.(metricdata.Gauge[int64])
	if !ok {
		return nil, fmt.Errorf("%s type gauge[float64] not found in %v", name, mtc)
	}

	for _, dataPoint := range gauge.DataPoints {
		if dataPoint.Attributes.Equals(attribs) {
			return &dataPoint, nil
		}
	}
	return nil, fmt.Errorf("%s named gauge[float64] with attribs %v not found in %v", name, attribs, mtc)
}

func (t *TestMetricsExporter) getNamedFloat64CounterData(name string, attribs *attribute.Set) (*metricdata.DataPoint[float64], error) {
	mtc, err := t.getNamedMetric(name)
	if err != nil {
		return nil, err
	}

	if counter, ok := mtc.Data.(metricdata.Sum[float64]); ok {
		for _, dataPoint := range counter.DataPoints {
			if dataPoint.Attributes.Equals(attribs) {
				return &dataPoint, nil
			}
		}
		return nil, fmt.Errorf("%s named counter[float64] with attribs %v not found in %v", name, attribs, mtc)
	}
	return nil, fmt.Errorf("%s type counter[float64] not found in %v", name, mtc)
}

func (t *TestMetricsExporter) getNamedFloat64HistogramData(name string, attribs *attribute.Set) (*metricdata.HistogramDataPoint[float64], error) {
	mtc, err := t.getNamedMetric(name)
	if err != nil {
		return nil, err
	}

	if histogram, ok := mtc.Data.(metricdata.Histogram[float64]); ok {
		for _, dataPoint := range histogram.DataPoints {
			if dataPoint.Attributes.Equals(attribs) {
				return &dataPoint, nil
			}
		}
		return nil, fmt.Errorf("%s named counter[float64] with attribs %v not found in %v", name, attribs, mtc)
	}
	return nil, fmt.Errorf("%s type counter[float64] not found in %v", name, mtc)
}

// GetFloat64HistogramData returns an otel histogram float64 data point for test reads
func (t *TestMetricsExporter) GetFloat64HistogramData(name string, attribs *attribute.Set) (*metricdata.HistogramDataPoint[float64], error) {
	data, err := t.getNamedFloat64HistogramData(name, attribs)
	return data, err
}

// GetInt64CounterValue returns an otel int64 counter value for test reads
func (t *TestMetricsExporter) GetInt64CounterValue(name string, attribs *attribute.Set) (int64, error) {
	counter, err := t.getNamedInt64CounterData(name, attribs)
	if err != nil {
		return 0, err
	}
	return counter.Value, err
}

// GetFloat64GaugeValue returns an otel float64 gauge value for test reads
func (t *TestMetricsExporter) GetFloat64GaugeValue(name string, attribs *attribute.Set) (float64, error) {
	gauge, err := t.getNamedFloat64GaugeData(name, attribs)
	if err != nil {
		return 0, err
	}
	return gauge.Value, err
}

// GetInt64GaugeValue returns an otel int64 gauge value for test reads
func (t *TestMetricsExporter) GetInt64GaugeValue(name string, attribs *attribute.Set) (int64, error) {
	gauge, err := t.getNamedInt64GaugeData(name, attribs)
	if err != nil {
		return 0, err
	}
	return gauge.Value, err
}

// GetFloat64CounterValue returns an otel float64 counter value for test reads
func (t *TestMetricsExporter) GetFloat64CounterValue(name string, attribs *attribute.Set) (float64, error) {
	counter, err := t.getNamedFloat64CounterData(name, attribs)
	if err != nil {
		return 0, err
	}
	return counter.Value, err
}
