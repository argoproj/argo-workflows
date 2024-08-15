package metrics

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"k8s.io/client-go/util/workqueue"
)

// TestExporter is an opentelemetry metrics exporter, purely for use within
// tests. It is not possible to query the values of an instrument via the otel
// SDK, so this exporter provides methods by which you can request
// metrics by name+attributes and therefore inspect whether they exist, and
// their values for the purposes of testing only.
// This is a public structure as it is used outside of this module also.
type TestExporter struct {
	metric.Reader
}

// TestScopeName is the name that the metrics running under test will have
const TestScopeName string = "argo-workflows-test"

var _ metric.Reader = &TestExporter{}

var sharedMetrics *Metrics = nil
var sharedTE *TestExporter = nil

// getSharedMetrics returns a singleton metrics with test exporter
// This is necessary because only the first call to workqueue.SetProvider
// takes effect within a single binary
// This can be fixed when we update to client-go 0.27 or later and we can
// create workqueues with https://godocs.io/k8s.io/client-go/util/workqueue#NewRateLimitingQueueWithConfig
func getSharedMetrics() (*Metrics, *TestExporter, error) {
	if sharedMetrics == nil {
		config := Config{
			Enabled: true,
			TTL:     1 * time.Second,
		}
		var err error
		sharedMetrics, sharedTE, err = createTestMetrics(&config, Callbacks{})
		if err != nil {
			return nil, nil, err
		}

		workqueue.SetProvider(sharedMetrics)
	}
	return sharedMetrics, sharedTE, nil
}

// CreateDefaultTestMetrics creates a boring testExporter enabled
// metrics, suitable for many tests
func CreateDefaultTestMetrics() (*Metrics, *TestExporter, error) {
	config := Config{
		Enabled: true,
	}
	return createTestMetrics(&config, Callbacks{})
}

func createTestMetrics(config *Config, callbacks Callbacks) (*Metrics, *TestExporter, error) {
	ctx /* with cancel*/ := context.Background()
	te := newTestExporter()

	m, err := New(ctx, TestScopeName, config, callbacks, metric.WithReader(te))
	return m, te, err

}

func newTestExporter() *TestExporter {
	reader := metric.NewManualReader()

	e := &TestExporter{
		Reader: reader,
	}
	return e
}

func (t *TestExporter) getOurMetrics() (*[]metricdata.Metrics, error) {
	metrics := metricdata.ResourceMetrics{}
	err := t.Collect(context.TODO(), &metrics)
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

func (t *TestExporter) getNamedMetric(name string) (*metricdata.Metrics, error) {
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

func (t *TestExporter) getNamedInt64CounterData(name string, attribs *attribute.Set) (*metricdata.DataPoint[int64], error) {
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

func (t *TestExporter) getNamedFloat64GaugeData(name string, attribs *attribute.Set) (*metricdata.DataPoint[float64], error) {
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

func (t *TestExporter) getNamedInt64GaugeData(name string, attribs *attribute.Set) (*metricdata.DataPoint[int64], error) {
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

func (t *TestExporter) getNamedFloat64CounterData(name string, attribs *attribute.Set) (*metricdata.DataPoint[float64], error) {
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

func (t *TestExporter) getNamedFloat64HistogramData(name string, attribs *attribute.Set) (*metricdata.HistogramDataPoint[float64], error) {
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
func (t *TestExporter) GetFloat64HistogramData(name string, attribs *attribute.Set) (*metricdata.HistogramDataPoint[float64], error) {
	data, err := t.getNamedFloat64HistogramData(name, attribs)
	return data, err
}

// GetInt64CounterValue returns an otel int64 counter value for test reads
func (t *TestExporter) GetInt64CounterValue(name string, attribs *attribute.Set) (int64, error) {
	counter, err := t.getNamedInt64CounterData(name, attribs)
	if err != nil {
		return 0, err
	}
	return counter.Value, err
}

// GetFloat64GaugeValue returns an otel float64 gauge value for test reads
func (t *TestExporter) GetFloat64GaugeValue(name string, attribs *attribute.Set) (float64, error) {
	gauge, err := t.getNamedFloat64GaugeData(name, attribs)
	if err != nil {
		return 0, err
	}
	return gauge.Value, err
}

// GetInt64GaugeValue returns an otel int64 gauge value for test reads
func (t *TestExporter) GetInt64GaugeValue(name string, attribs *attribute.Set) (int64, error) {
	gauge, err := t.getNamedInt64GaugeData(name, attribs)
	if err != nil {
		return 0, err
	}
	return gauge.Value, err
}

// GetFloat64CounterValue returns an otel float64 counter value for test reads
func (t *TestExporter) GetFloat64CounterValue(name string, attribs *attribute.Set) (float64, error) {
	counter, err := t.getNamedFloat64CounterData(name, attribs)
	if err != nil {
		return 0, err
	}
	return counter.Value, err
}
