package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/client-go/util/workqueue"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func TestMetricsWorkQueue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	m, te, err := getSharedMetrics(ctx)
	require.NoError(t, err)

	attribsWT := attribute.NewSet(attribute.String(telemetry.AttribWorkerType, "test"))

	queue := m.RateLimiterWithBusyWorkers(ctx, workqueue.DefaultTypedControllerRateLimiter[string](), "test")
	defer queue.ShutDown()
	val, err := te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkersBusyCount.Name(), &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	attribsQN := attribute.NewSet(attribute.String(telemetry.AttribQueueName, "test"))
	queue.Add("A")
	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkersBusyCount.Name(), &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentQueueDepthGauge.Name(), &attribsQN)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	queue.Get()
	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkersBusyCount.Name(), &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentQueueDepthGauge.Name(), &attribsQN)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	queue.Done("A")
	val, err = te.GetInt64CounterValue(ctx, telemetry.InstrumentWorkersBusyCount.Name(), &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)
}
