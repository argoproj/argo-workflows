package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/util/workqueue"
)

func TestMetricsWorkQueue(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	require.Empty(t, m.workersBusy)

	m.newWorker("test")
	require.Len(t, m.workersBusy, 1)
	require.InDelta(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value, 0.001)

	m.newWorker("test")
	require.Len(t, m.workersBusy, 1)

	queue := m.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "test")
	defer queue.ShutDown()

	queue.Add("A")
	require.InDelta(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value, 0.001)

	queue.Get()
	require.InDelta(t, float64(1), *write(m.workersBusy["test"]).Gauge.Value, 0.001)

	queue.Done("A")
	require.InDelta(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value, 0.001)
}
