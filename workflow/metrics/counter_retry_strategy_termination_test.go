package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func TestRecordRetryStrategyTermination(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, exporter, err := CreateDefaultTestMetrics(ctx)
	require.NoError(t, err)
	require.NotNil(t, m.GetInstrument(telemetry.InstrumentRetryStrategyTerminationsTotal.Name()))

	const namespace = "argo"
	tests := []struct {
		name   string
		reason RetryStrategyTerminationReason
	}{
		{
			name:   "max duration exceeded",
			reason: RetryStrategyTerminationReasonMaxDurationExceeded,
		},
		{
			name:   "backoff would exceed max duration",
			reason: RetryStrategyTerminationReasonBackoffWouldExceedMaxDuration,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.RecordRetryStrategyTermination(ctx, tt.reason, namespace)

			attributes := attribute.NewSet(
				attribute.String(telemetry.AttribRetryStrategyTerminationReason, string(tt.reason)),
				attribute.String(telemetry.AttribWorkflowNamespace, namespace),
			)
			value, err := exporter.GetInt64CounterValue(ctx, telemetry.InstrumentRetryStrategyTerminationsTotal.Name(), &attributes)
			require.NoError(t, err)
			assert.Equal(t, int64(1), value)
		})
	}
}
