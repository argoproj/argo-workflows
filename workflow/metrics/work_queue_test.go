package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/util/workqueue"
)

func TestMetricsWorkQueue(t *testing.T) {
	config := ServerConfig{
		Enabled: true,
		Path:    DefaultMetricsServerPath,
		Port:    DefaultMetricsServerPort,
	}
	m := New(config, config)

	assert.Empty(t, m.workersBusy)

	m.newWorker("test")
	assert.Len(t, m.workersBusy, 1)
	assert.Equal(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value)

	m.newWorker("test")
	assert.Len(t, m.workersBusy, 1)

	queue := m.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "test")
	defer queue.ShutDown()

	queue.Add("A")
	assert.Equal(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value)

	queue.Get()
	assert.Equal(t, float64(1), *write(m.workersBusy["test"]).Gauge.Value)

	queue.Done("A")
	assert.Equal(t, float64(0), *write(m.workersBusy["test"]).Gauge.Value)
}
