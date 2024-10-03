package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/sdk/metric"
)

func createDefaultTestMetrics() (*Metrics, *TestMetricsExporter, error) {
	config := Config{
		Enabled: true,
	}
	return createTestMetrics(&config)
}

func createTestMetrics(config *Config) (*Metrics, *TestMetricsExporter, error) {
	ctx /* with cancel*/ := context.Background()
	te := NewTestMetricsExporter()

	m, err := NewMetrics(ctx, TestScopeName, TestScopeName, config, metric.WithReader(te))
	if err != nil {
		return nil, nil, err
	}
	err = m.Populate(ctx, AddVersion, addTestingCounter, addTestingHistogram)
	return m, te, err
}

const (
	nameTestingHistogram = `testing_histogram`
	nameTestingCounter   = `testing_counter`
	errorCauseTestingA   = "TestingA"
	errorCauseTestingB   = "TestingB"
)

func addTestingHistogram(_ context.Context, m *Metrics) error {
	// The buckets here are only the 'defaults' and can be overridden with configmap defaults
	return m.CreateInstrument(Float64Histogram,
		nameTestingHistogram,
		"Testing Metric",
		"s",
		WithDefaultBuckets([]float64{0.0, 1.0, 5.0, 10.0}),
		WithAsBuiltIn(),
	)
}

func (m *Metrics) TestingHistogramRecord(ctx context.Context, value float64) {
	m.Record(ctx, nameTestingHistogram, value, InstAttribs{})
}

func addTestingCounter(ctx context.Context, m *Metrics) error {
	return m.CreateInstrument(Int64Counter,
		nameTestingCounter,
		"Testing Error Counting Metric",
		"{errors}",
		WithAsBuiltIn(),
	)
}

func (m *Metrics) TestingErrorA(ctx context.Context) {
	m.AddInt(ctx, nameTestingCounter, 1, InstAttribs{{Name: AttribErrorCause, Value: errorCauseTestingB}})
}

func (m *Metrics) TestingErrorB(ctx context.Context) {
	m.AddInt(ctx, nameTestingCounter, 1, InstAttribs{{Name: AttribErrorCause, Value: errorCauseTestingB}})
}
