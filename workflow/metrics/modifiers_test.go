package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func TestViewDisable(t *testing.T) {
	// Same metric as TestMetrics, but disabled by a view
	m, te, err := createTestMetrics(&Config{
		Modifiers: map[string]Modifier{
			nameOperationDuration: {
				Disabled: true,
			},
		},
	},
		Callbacks{},
	)
	assert.NoError(t, err)
	m.OperationCompleted(m.ctx, 5)
	attribs := attribute.NewSet()
	_, err = te.GetFloat64HistogramData(nameOperationDuration, &attribs)
	assert.Error(t, err)
}

func TestViewDisabledAttributes(t *testing.T) {
	// Disable the error cause label
	m, te, err := createTestMetrics(&Config{
		Modifiers: map[string]Modifier{
			nameErrorCount: {
				DisabledAttributes: []string{labelErrorCause},
			},
		},
	},
		Callbacks{},
	)
	assert.NoError(t, err)
	// Submit a couple of errors
	m.OperationPanic(context.Background())
	m.CronWorkflowSubmissionError(context.Background())
	// See if we can find this with the attributes, we should not be able to
	attribsFail := attribute.NewSet(attribute.String(labelErrorCause, string(ErrorCauseOperationPanic)))
	_, err = te.GetInt64CounterValue(nameErrorCount, &attribsFail)
	assert.Error(t, err)
	// Find a sum of all error types
	attribsSuccess := attribute.NewSet()
	val, err := te.GetInt64CounterValue(nameErrorCount, &attribsSuccess)
	assert.NoError(t, err)
	// Sum of the two submitted errors is 2
	assert.Equal(t, int64(2), val)
}

func TestViewHistogramBuckets(t *testing.T) {
	// Same metric as TestMetrics, but buckets changed
	bounds := []float64{1.0, 3.0, 5.0, 10.0}
	m, te, err := createTestMetrics(&Config{
		Modifiers: map[string]Modifier{
			nameOperationDuration: {
				HistogramBuckets: bounds,
			},
		},
	},
		Callbacks{},
	)
	assert.NoError(t, err)
	m.OperationCompleted(m.ctx, 5)
	attribs := attribute.NewSet()
	val, err := te.GetFloat64HistogramData(nameOperationDuration, &attribs)
	assert.NoError(t, err)
	assert.Equal(t, bounds, val.Bounds)
	assert.Equal(t, []uint64{0, 0, 1, 0, 0}, val.BucketCounts)
}
