package metrics

import (
	"k8s.io/client-go/util/workqueue"
)

type workerBusyWorkQueueDecorator struct {
	workqueue.RateLimitingInterface
	name string
}

func NewWorkQueue(x workqueue.RateLimiter, name string) workqueue.RateLimitingInterface {
	WorkersBusyMetric.WithLabelValues(name).Set(0)
	return workerBusyWorkQueueDecorator{workqueue.NewNamedRateLimitingQueue(x, name), name}
}

func (b workerBusyWorkQueueDecorator) Get() (interface{}, bool) {
	item, shutdown := b.RateLimitingInterface.Get()
	WorkersBusyMetric.WithLabelValues(b.name).Inc()
	return item, shutdown
}

func (b workerBusyWorkQueueDecorator) Done(item interface{}) {
	b.RateLimitingInterface.Done(item)
	WorkersBusyMetric.WithLabelValues(b.name).Dec()
}
