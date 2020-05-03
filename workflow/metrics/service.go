package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Service interface {
	Metrics() []prometheus.Metric
	InsignificantPodChange()
	PodResourceVersionRepeated()
	PodProcessed()
	SignificantPodChange()
	WorkflowResourceVersionRepeated()
	WorkflowProcessed(duration time.Duration)
	UpdatesPersisted()
	UpdateReapplied()
}

func NewService() Service {
	return &service{
		insignificantPodChange:          prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "insignificant_pod_update", Help: "Number of insignificant pod updates"}),
		updatesPersisted:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_persisted", Help: "Number of times we persisted a workflow update"}),
		updatesReapplied:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_reapplied", Help: "Number of times we re-applied a workflow update. Ideally should always be zero."}),
		podResourceVersionRepeated:      prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pod_resource_version_repeated", Help: "Number of pod updates had the same resource version as the old one"}),
		podsProcessed:                   prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pods_processed", Help: "Number of pod updates processed"}),
		significantPodChange:            prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "significant_pod_update", Help: "Number of significant pod updates"}),
		workflowResourceVersionRepeated: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflow_resource_version_repeated", Help: "Number of workflow updates that the same resource version as the old one"}),
		workflowsProcessed: prometheus.NewHistogram(prometheus.HistogramOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflows_processed", Help: "Workflow updates processed",
			Buckets: prometheus.ExponentialBuckets(float64(10*time.Millisecond), float64(2), 8)},
		),
	}
}

type service struct {
	insignificantPodChange          prometheus.Counter
	updatesPersisted                prometheus.Counter
	updatesReapplied                prometheus.Counter
	podResourceVersionRepeated      prometheus.Counter
	podsProcessed                   prometheus.Counter
	significantPodChange            prometheus.Counter
	workflowsProcessed              prometheus.Histogram
	workflowResourceVersionRepeated prometheus.Counter
}

func (s *service) InsignificantPodChange() {
	s.insignificantPodChange.Inc()
}

func (s *service) Metrics() []prometheus.Metric {
	return []prometheus.Metric{
		s.insignificantPodChange,
		s.updatesPersisted,
		s.updatesReapplied,
		s.podResourceVersionRepeated,
		s.podsProcessed,
		s.significantPodChange,
		s.workflowsProcessed,
		s.workflowResourceVersionRepeated,
	}
}

func (s *service) PodResourceVersionRepeated() {
	s.podResourceVersionRepeated.Inc()
}

func (s *service) PodProcessed() {
	s.podsProcessed.Inc()
}

func (s *service) SignificantPodChange() {
	s.significantPodChange.Inc()
}

func (s *service) UpdatesPersisted() {
	s.updatesPersisted.Inc()
}

func (s *service) UpdateReapplied() {
	s.updatesReapplied.Inc()
}

func (s *service) WorkflowResourceVersionRepeated() {
	s.workflowResourceVersionRepeated.Inc()
}

func (s *service) WorkflowProcessed(duration time.Duration) {
	s.workflowsProcessed.Observe(float64(duration))
}
