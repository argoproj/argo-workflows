package metrics

import "k8s.io/client-go/util/workqueue"

type workersBusyRateLimiterWorkQueue struct {
	workqueue.RateLimitingInterface
	queueName string
	metrics   *Metrics
}

func (m *Metrics) RateLimiterWithBusyWorkers(workQueue workqueue.RateLimiter, queueName string) workqueue.RateLimitingInterface {
	m.newWorker(queueName)
	return workersBusyRateLimiterWorkQueue{
		RateLimitingInterface: workqueue.NewNamedRateLimitingQueue(workQueue, queueName),
		queueName:             queueName,
		metrics:               m,
	}
}

func (m *Metrics) newWorker(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, ok := m.workersBusy[workerType]; !ok {
		m.workersBusy[workerType] = getWorkersBusy(workerType)
	}
}

func (m *Metrics) workerBusy(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric, ok := m.workersBusy[workerType]; ok {
		metric.Inc()
	}
}

func (m *Metrics) workerFree(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric, ok := m.workersBusy[workerType]; ok {
		metric.Dec()
	}
}

func (w workersBusyRateLimiterWorkQueue) Get() (interface{}, bool) {
	item, shutdown := w.RateLimitingInterface.Get()
	w.metrics.workerBusy(w.queueName)
	return item, shutdown

}

func (w workersBusyRateLimiterWorkQueue) Done(item interface{}) {
	w.RateLimitingInterface.Done(item)
	w.metrics.workerFree(w.queueName)
}
