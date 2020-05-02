package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Service interface {
	UpdateReapplied()
	UpdatesPersisted()
	Metrics() []prometheus.Metric
	PodProcessed()
	WorkflowProcessed(duration time.Duration)
}

type service struct {
	updatesPersisted   prometheus.Counter
	updatesReapplied   prometheus.Counter
	podsProcessed      prometheus.Counter
	workflowsProcessed prometheus.Histogram
}

func (s *service) Metrics() []prometheus.Metric {
	return []prometheus.Metric{s.updatesPersisted, s.updatesReapplied, s.podsProcessed, s.workflowsProcessed}
}

func (s *service) WorkflowProcessed(duration time.Duration) {
	s.workflowsProcessed.Observe(float64(duration))
}

func (s *service) PodProcessed() {
	s.podsProcessed.Inc()
}

func (s *service) UpdatesPersisted() {
	s.updatesPersisted.Inc()
}

func (s *service) UpdateReapplied() {
	s.updatesReapplied.Inc()
}

func NewService() Service {
	return &service{
		updatesPersisted: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_persisted", Help: "Number of times we persisted a workflow update"}),
		updatesReapplied: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_reapplied", Help: "Number of times we had to re-apply a workflow update"}),
		podsProcessed:    prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pods_processed", Help: "Pod changes processed"}),
		workflowsProcessed: prometheus.NewHistogram(prometheus.HistogramOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflows_processed", Help: "Workflow changes processed",
			Buckets: prometheus.ExponentialBuckets(float64(50), float64(2), 8)},
		),
	}
}
