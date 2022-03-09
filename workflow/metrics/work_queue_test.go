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

	assert.Len(t, m.workersBusy, 0)

	m.newWorker("test")
	assert.Len(t, m.workersBusy, 1)

	m.newWorker("test")
	assert.Len(t, m.workersBusy, 1)

	queue := m.RateLimiterWithBusyWorkers(workqueue.DefaultControllerRateLimiter(), "test")
	defer queue.ShutDown()

	queue.Add("A")

	queue.Get()

	queue.Done("A")
}
