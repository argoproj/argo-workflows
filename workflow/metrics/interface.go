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
	WorkflowUpdated(from, to wfv1.NodePhase)
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
}

func New() Interface {
	return &metrics{
		insignificantPodChange:          prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "insignificant_pod_update", Help: "Number of insignificant pod updates"}),
		updatesPersisted:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_persisted", Help: "Number of times we persisted a workflow update"}),
		updatesReapplied:                prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "updates_reapplied", Help: "Number of times we re-applied a workflow update. Ideally should always be zero."}),
		podResourceVersionRepeated:      prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pod_resource_version_repeated", Help: "Number of pod updates had the same resource version as the old one"}),
		podsProcessed:                   prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "pods_processed", Help: "Number of pod updates processed"}),
		significantPodChange:            prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "significant_pod_update", Help: "Number of significant pod updates"}),
		workflowResourceVersionRepeated: prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflow_resource_version_repeated", Help: "Number of workflow updates that the same resource version as the old one"}),
		workflowsProcessed:              prometheus.NewCounter(prometheus.CounterOpts{Namespace: argoNamespace, Subsystem: workflowsSubsystem, Name: "workflows_processed", Help: "Number of workflow updates processed"}),
		nodePhases: map[wfv1.NodePhase]prometheus.Gauge{
			wfv1.NodePending:   prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodePending)),
			wfv1.NodeRunning:   prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodeRunning)),
			wfv1.NodeSucceeded: prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodeSucceeded)),
			wfv1.NodeSkipped:   prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodeSkipped)),
			wfv1.NodeFailed:    prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodeFailed)),
			wfv1.NodeError:     prometheus.NewGauge(getPhaseGaugeOpts(wfv1.NodeError)),
		},
		custom: make(map[string]common.Metric),
	}
}

func getPhaseGaugeOpts(phase wfv1.NodePhase) prometheus.GaugeOpts {
	return prometheus.GaugeOpts{
		Namespace:   argoNamespace,
		Subsystem:   workflowsSubsystem,
		Name:        "count",
		Help:        "Number of Workflows currently accessible by the controller by status",
		ConstLabels: map[string]string{"status": string(phase)},
	}
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
	nodePhases                      map[wfv1.NodePhase]prometheus.Gauge
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

func (m *metrics) WorkflowUpdated(from, to wfv1.NodePhase) {
	m.nodePhases[from].Dec()
	m.nodePhases[to].Dec()
}

func (m *metrics) InsignificantPodChange() {
	m.insignificantPodChange.Inc()
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
	for _, gauge := range m.nodePhases {
		metrics = append(metrics, gauge)
	}
	for _, metric := range m.custom {
		metrics = append(metrics, metric)
	}
	return metrics
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
