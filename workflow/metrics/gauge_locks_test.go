package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func lockAttribs(lockType, storage, name, namespace string) attribute.Set {
	return attribute.NewSet(
		attribute.String("type", lockType),
		attribute.String("storage", storage),
		attribute.String("lock_name", name),
		attribute.String("namespace", namespace),
	)
}

func TestLocksGauges(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := createTestMetrics(ctx, &telemetry.MetricsConfig{}, Callbacks{})
	require.NoError(t, err)
	require.NoError(t, m.RegisterLockGauges(func(_ context.Context) []LockGaugeSample {
		return []LockGaugeSample{
			{Type: "mutex", Storage: "configmap", Name: "my-mutex", Namespace: "default", Held: 1, Pending: 2},
			{Type: "semaphore", Storage: "database", Name: "my-sem", Namespace: "ns2", Held: 3, Pending: 0},
		}
	}))

	mutexAttribs := lockAttribs("mutex", "configmap", "my-mutex", "default")
	held, err := te.GetInt64GaugeValue(ctx, telemetry.InstrumentLocksHeld.Name(), &mutexAttribs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), held, "mutex held")
	pending, err := te.GetInt64GaugeValue(ctx, telemetry.InstrumentLocksPending.Name(), &mutexAttribs)
	require.NoError(t, err)
	assert.Equal(t, int64(2), pending, "mutex pending")

	semAttribs := lockAttribs("semaphore", "database", "my-sem", "ns2")
	held, err = te.GetInt64GaugeValue(ctx, telemetry.InstrumentLocksHeld.Name(), &semAttribs)
	require.NoError(t, err)
	assert.Equal(t, int64(3), held, "semaphore held")
}

func TestRecordLockTaken(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := createTestMetrics(ctx, &telemetry.MetricsConfig{}, Callbacks{})
	require.NoError(t, err)

	m.RecordLockTaken(ctx, "semaphore", "database", "my-sem", "default")
	m.RecordLockTaken(ctx, "semaphore", "database", "my-sem", "default")

	attribs := lockAttribs("semaphore", "database", "my-sem", "default")
	val, err := te.GetInt64CounterValue(ctx, telemetry.InstrumentLocksTakenTotal.Name(), &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(2), val)
}
