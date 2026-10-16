package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestCompletionRequeueMetricHelpers(t *testing.T) {
	for _, tt := range []struct {
		instrument BuiltinInstrument
		record     func(*Metrics, context.Context, float64)
		value      float64
		buckets    []float64
	}{
		{InstrumentWorkflowCompletionRequeueDelay, (*Metrics).RecordWorkflowCompletionRequeueDelay, 0.25,
			[]float64{0.001, 0.01, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30}},
		{InstrumentWorkflowReconcileUtilization, (*Metrics).RecordWorkflowReconcileUtilization, 0.2,
			[]float64{0.01, 0.05, 0.1, 0.2, 0.3, 0.5, 0.75, 0.9, 1}},
	} {
		t.Run(tt.instrument.Name(), func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			m, exporter, err := createDefaultTestMetrics(ctx)
			require.NoError(t, err)
			defer func() { assert.NoError(t, m.Shutdown(ctx)) }()
			require.NoError(t, m.CreateBuiltinInstrument(tt.instrument))
			tt.record(m, ctx, tt.value)
			attrs := attribute.NewSet()
			data, err := exporter.GetFloat64HistogramData(ctx, tt.instrument.Name(), &attrs)
			require.NoError(t, err)
			assert.Equal(t, uint64(1), data.Count)
			assert.InDelta(t, tt.value, data.Sum, 0.000001)
			assert.Equal(t, tt.buckets, data.Bounds)
			assert.Zero(t, data.Attributes.Len())
		})
	}
}
