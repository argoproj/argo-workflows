package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/client-go/util/workqueue"
)

func TestMetricsWorkQueue(t *testing.T) {
	m, te, err := getSharedMetrics()
	require.NoError(t, err)

	attribsWT := attribute.NewSet(attribute.String(labelWorkerType, "test"))

	queue := m.RateLimiterWithBusyWorkers(m.ctx, workqueue.DefaultControllerRateLimiter(), "test")
	defer queue.ShutDown()
	val, err := te.GetInt64CounterValue(nameWorkersBusy, &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	attribsQN := attribute.NewSet(attribute.String(labelQueueName, "test"))
	queue.Add("A")
	val, err = te.GetInt64CounterValue(nameWorkersBusy, &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	val, err = te.GetInt64CounterValue(nameWorkersQueueDepth, &attribsQN)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)

	queue.Get()
	val, err = te.GetInt64CounterValue(nameWorkersBusy, &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
	val, err = te.GetInt64CounterValue(nameWorkersQueueDepth, &attribsQN)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)

	queue.Done("A")
	val, err = te.GetInt64CounterValue(nameWorkersBusy, &attribsWT)
	require.NoError(t, err)
	assert.Equal(t, int64(0), val)
}
