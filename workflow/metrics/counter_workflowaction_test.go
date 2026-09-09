package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func TestWorkflowActionProcessedCounter(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := createTestMetrics(ctx, &telemetry.MetricsConfig{}, Callbacks{})
	require.NoError(t, err)

	m.WorkflowActionProcessed(ctx, "Terminate", "Succeeded", "argo")
	m.WorkflowActionProcessed(ctx, "Terminate", "Succeeded", "argo")
	m.WorkflowActionProcessed(ctx, "Resume", "Failed", "other")

	attribs := attribute.NewSet(
		attribute.String("action", "Terminate"),
		attribute.String("outcome", "Succeeded"),
		attribute.String("namespace", "argo"),
	)
	val, err := te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkflowactionsProcessedTotal.Name(), &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(2), val)

	attribs = attribute.NewSet(
		attribute.String("action", "Resume"),
		attribute.String("outcome", "Failed"),
		attribute.String("namespace", "other"),
	)
	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkflowactionsProcessedTotal.Name(), &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}
