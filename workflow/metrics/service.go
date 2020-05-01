package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Service interface {
	MetricsProvider
	ReapplyUpdate() prometheus.Counter
	PodChange(significant bool) prometheus.Counter
}

type service struct {
	reapplyUpdate          prometheus.Counter
	significantPodChange   prometheus.Counter
	insignificantPodChange prometheus.Counter
}

func (s *service) DeleteExpiredMetrics(time.Duration) {
	// these never expire
}

func (s *service) GetMetrics() []prometheus.Metric {
	return []prometheus.Metric{
		s.reapplyUpdate,
		s.significantPodChange,
		s.insignificantPodChange,
	}
}

func (s *service) PodChange(significant bool) prometheus.Counter {
	if significant {
		return s.significantPodChange
	}
	return s.insignificantPodChange
}

func (s *service) ReapplyUpdate() prometheus.Counter {
	return s.reapplyUpdate
}

func NewService() Service {
	return &service{
		reapplyUpdate:          prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "reapply_update", Help: "Number of times we had to re-apply a workflow update"}),
		significantPodChange:   prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "significant_pod_change", Help: "Number of times a pod change was significant"}),
		insignificantPodChange: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "insignificant_pod_change", Help: "Number of time a pod change was insignificant"}),
	}
}
