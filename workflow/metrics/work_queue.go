package metrics

import "k8s.io/client-go/util/workqueue"

type workersBusyRateLimiterWorkQueue struct {
	workqueue.RateLimitingInterface
	workerType string
	metrics    *Metrics
}

func (m *Metrics) RateLimiterWithBusyWorkers(workQueue workqueue.RateLimiter, queueName string) workqueue.RateLimitingInterface {
	m.newWorker(queueName)
	return workersBusyRateLimiterWorkQueue{
		RateLimitingInterface: workqueue.NewNamedRateLimitingQueue(workQueue, queueName),
		workerType:            queueName,
		metrics:               m,
	}
}

func (m *Metrics) newWorker(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.workersBusy[workerType] = getWorkersBusy(workerType)
}

func (m *Metrics) workerBusy(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.workersBusy[workerType].Inc()
}

func (m *Metrics) workerFree(workerType string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.workersBusy[workerType].Dec()
}

func (w workersBusyRateLimiterWorkQueue) Get() (interface{}, bool) {
	item, shutdown := w.RateLimitingInterface.Get()
	w.metrics.workerBusy(w.workerType)
	return item, shutdown
}

func (w workersBusyRateLimiterWorkQueue) Done(item interface{}) {
	w.RateLimitingInterface.Done(item)
	w.metrics.workerFree(w.workerType)
}
