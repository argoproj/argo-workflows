package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

type Interface interface {
	prometheus.Collector
	// the following funcs are named after events that can happen and therefore all match the commond naming convention
	// for events - i.e. name+verb(past tense)
	InsignificantPodChange()
	PodResourceVersionRepeated()
	PodProcessed()
	SignificantPodChange()
	WorkflowResourceVersionRepeated()
	WorkflowProcessed()
	UpdatesPersisted()
	UpdateReapplied()
	// delete any expired custom metrics
	DeleteExpiredMetrics(ttl time.Duration)
	SetCustom(desc string, metric common.Metric)
	GetCustom(desc string) prometheus.Metric
	WorkflowAdded(phase wfv1.NodePhase)
	WorkflowUpdated(from, to wfv1.NodePhase)
	WorkflowDeleted(phase wfv1.NodePhase)
}

func New() Interface {
	return &metrics{
		insignificantPodChange:          prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pod_updates", Help: "Number of pod updates", ConstLabels: map[string]string{"significant": "false"}}),
		significantPodChange:            prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pod_updates", Help: "Number of pod updates", ConstLabels: map[string]string{"significant": "true"}}),
		updatesPersisted:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_persisted", Help: "Number of times we persisted a workflow update"}),
		updatesReapplied:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_reapplied", Help: "Number of times we re-applied a workflow update. Ideally should always be zero."}),
		podResourceVersionRepeated:      prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pod_resource_version_repeated", Help: "Number of pod updates had the same resource version as the old one"}),
		podsProcessed:                   prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pods_processed", Help: "Number of pod updates processed"}),
		workflowResourceVersionRepeated: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflow_resource_version_repeated", Help: "Number of workflow updates that the same resource version as the old one"}),
		workflowsProcessed:              prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflows_processed", Help: "Number of workflow updates processed"}),
		runtimePhases: map[wfv1.NodePhase]prometheus.Gauge{
			wfv1.NodePending:   prometheus.NewGauge(runtimePhaseOpts(wfv1.NodePending)),
			wfv1.NodeRunning:   prometheus.NewGauge(runtimePhaseOpts(wfv1.NodeRunning)),
			wfv1.NodeSucceeded: prometheus.NewGauge(runtimePhaseOpts(wfv1.NodeSucceeded)),
			wfv1.NodeSkipped:   prometheus.NewGauge(runtimePhaseOpts(wfv1.NodeSkipped)),
			wfv1.NodeFailed:    prometheus.NewGauge(runtimePhaseOpts(wfv1.NodeFailed)),
			wfv1.NodeError:     prometheus.NewGauge(runtimePhaseOpts(wfv1.NodeError)),
		},
		completedPhases: map[wfv1.NodePhase]prometheus.Counter{
			wfv1.NodeSucceeded: prometheus.NewCounter(completedPhaseOpts(wfv1.NodeSucceeded)),
			wfv1.NodeSkipped:   prometheus.NewCounter(completedPhaseOpts(wfv1.NodeSkipped)),
			wfv1.NodeFailed:    prometheus.NewCounter(completedPhaseOpts(wfv1.NodeFailed)),
			wfv1.NodeError:     prometheus.NewCounter(completedPhaseOpts(wfv1.NodeError)),
		},
		custom: make(map[string]common.Metric),
	}
}

func runtimePhaseOpts(phase wfv1.NodePhase) prometheus.GaugeOpts {
	return prometheus.GaugeOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "count", Help: "Number of Workflows currently accessible by the controller by status", ConstLabels: map[string]string{"status": string(phase)}}
}

func completedPhaseOpts(phase wfv1.NodePhase) prometheus.CounterOpts {
	return prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "complete", Help: "Number of completed workflows by status", ConstLabels: map[string]string{"status": string(phase)}}
}

type metrics struct {
	insignificantPodChange          prometheus.Counter
	updatesPersisted                prometheus.Counter
	updatesReapplied                prometheus.Counter
	podResourceVersionRepeated      prometheus.Counter
	podsProcessed                   prometheus.Counter
	significantPodChange            prometheus.Counter
	workflowsProcessed              prometheus.Counter
	workflowResourceVersionRepeated prometheus.Counter
	runtimePhases                   map[wfv1.NodePhase]prometheus.Gauge
	completedPhases                 map[wfv1.NodePhase]prometheus.Counter
	custom                          map[string]common.Metric
}

func (m metrics) GetCustom(desc string) prometheus.Metric {
	return m.custom[desc]
}

func (m metrics) SetCustom(desc string, metric common.Metric) {
	m.custom[desc] = metric
}

func (m metrics) Describe(descs chan<- *prometheus.Desc) {
	for _, metric := range m.metrics() {
		descs <- metric.Desc()
	}
}

func (m metrics) Collect(c chan<- prometheus.Metric) {
	for _, metric := range m.metrics() {
		c <- metric
	}
}

func (m *metrics) metrics() []prometheus.Metric {
	metrics := []prometheus.Metric{
		m.insignificantPodChange,
		m.updatesPersisted,
		m.updatesReapplied,
		m.podResourceVersionRepeated,
		m.podsProcessed,
		m.significantPodChange,
		m.workflowsProcessed,
		m.workflowResourceVersionRepeated,
	}
	for _, gauge := range m.runtimePhases {
		metrics = append(metrics, gauge)
	}
	for _, gauge := range m.completedPhases {
		metrics = append(metrics, gauge)
	}
	for _, metric := range m.custom {
		metrics = append(metrics, metric)
	}
	return metrics
}

func (m metrics) WorkflowAdded(phase wfv1.NodePhase) {
	m.runtimePhases[phase].Inc()
}

func (m *metrics) WorkflowUpdated(from, to wfv1.NodePhase) {
	m.runtimePhases[from].Dec()
	m.runtimePhases[to].Inc()
	if !from.Completed() && to.Completed() {
		m.completedPhases[to].Inc()
	}
}

func (m metrics) WorkflowDeleted(phase wfv1.NodePhase) {
	m.runtimePhases[phase].Dec()
}

func (m *metrics) InsignificantPodChange() {
	m.insignificantPodChange.Inc()
}

func (m *metrics) PodResourceVersionRepeated() {
	m.podResourceVersionRepeated.Inc()
}

func (m *metrics) PodProcessed() {
	m.podsProcessed.Inc()
}

func (m *metrics) SignificantPodChange() {
	m.significantPodChange.Inc()
}

func (m *metrics) UpdatesPersisted() {
	m.updatesPersisted.Inc()
}

func (m *metrics) UpdateReapplied() {
	m.updatesReapplied.Inc()
}

func (m *metrics) WorkflowResourceVersionRepeated() {
	m.workflowResourceVersionRepeated.Inc()
}

func (m *metrics) WorkflowProcessed() {
	m.workflowsProcessed.Inc()
}

func (m *metrics) DeleteExpiredMetrics(ttl time.Duration) {
	for key, metric := range m.custom {
		if time.Since(metric.LastUpdated) > ttl {
			delete(m.custom, key)
		}
	}
}
