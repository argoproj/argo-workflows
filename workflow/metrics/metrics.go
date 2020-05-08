package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/argoproj/argo/pkg/apis/workflow"
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/common"
)

const (
	argoNamespace      = "argo"
	workflowsSubsystem = "workflows"
)

type ServerConfig struct {
	Path string
	Port string
	TTL  time.Duration
}

type Metrics struct {
	registry     *prometheus.Registry
	serverConfig ServerConfig

	updatesReapplied                prometheus.Counter
	podResourceVersionRepeated      prometheus.Counter
	podsProcessed                   prometheus.Counter
	workflowResourceVersionRepeated prometheus.Counter
	workflowsProcessed              prometheus.Counter
	workflowsByPhase                map[v1alpha1.NodePhase]prometheus.Gauge
	customMetrics                   map[string]common.Metric
	updatesPersisted                map[string]prometheus.Counter
	podChange                       map[bool]prometheus.Counter
}

var _ prometheus.Collector = Metrics{}

func New(config ServerConfig) Metrics {
	metrics := Metrics{
		serverConfig:                    config,
		updatesReapplied:                newCounter("updates_reapplied", "Number of times we re-applied a workflow update. Ideally should always be zero.", nil),
		podResourceVersionRepeated:      newCounter("pod_resource_version_repeated", "Number of pod updates had the same resource version as the old one", nil),
		podsProcessed:                   newCounter("pods_processed", "Number of pod updates processed", nil),
		workflowResourceVersionRepeated: newCounter("workflow_resource_version_repeated", "Number of workflow updates that have the same resource version as the old one", nil),
		workflowsProcessed:              newCounter("workflows_processed", "Number of workflow updates processed", nil),
		workflowsByPhase:                getWorkflowPhaseGauges(),
		customMetrics:                   make(map[string]common.Metric),
		updatesPersisted: map[string]prometheus.Counter{
			workflow.WorkflowKind:                newCounter("updates_persisted", "Number of times an update was persisted", map[string]string{"kind": "workflow"}),
			workflow.CronWorkflowKind:            newCounter("updates_persisted", "Number of times an update was persisted", map[string]string{"kind": "cron_workflow"}),
			workflow.WorkflowTemplateKind:        newCounter("updates_persisted", "Number of times an update was persisted", map[string]string{"kind": "workflow_template"}),
			workflow.ClusterWorkflowTemplateKind: newCounter("updates_persisted", "Number of times an update was persisted", map[string]string{"kind": "cluster_workflow_template"}),
		},
		podChange: map[bool]prometheus.Counter{
			true:  newCounter("pod_updates", "Number of pod updates", map[string]string{"significant": "false"}),
			false: newCounter("pod_updates", "Number of pod updates", map[string]string{"significant": "true"}),
		},
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(metrics)
	registry.MustRegister(prometheus.NewGoCollector())
	metrics.registry = registry

	return metrics
}

func (m Metrics) allMetrics() []prometheus.Metric {
	allMetrics := []prometheus.Metric{
		m.updatesReapplied,
		m.podResourceVersionRepeated,
		m.podsProcessed,
		m.workflowResourceVersionRepeated,
		m.workflowsProcessed,
	}
	for _, metric := range m.workflowsByPhase {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.updatesPersisted {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.podChange {
		allMetrics = append(allMetrics, metric)
	}
	for _, metric := range m.customMetrics {
		allMetrics = append(allMetrics, metric.Metric)
	}

	return allMetrics
}

func (m Metrics) WorkflowAdded(phase v1alpha1.NodePhase) {
	if _, ok := m.workflowsByPhase[phase]; ok {
		m.workflowsByPhase[phase].Inc()
	}
}

func (m Metrics) WorkflowUpdated(fromPhase, toPhase v1alpha1.NodePhase) {
	m.WorkflowDeleted(fromPhase)
	m.WorkflowAdded(toPhase)
}

func (m Metrics) WorkflowDeleted(phase v1alpha1.NodePhase) {
	if _, ok := m.workflowsByPhase[phase]; ok {
		m.workflowsByPhase[phase].Dec()
	}
}

func (m Metrics) GetCustomMetric(key string) common.Metric {
	// It's okay to return nil metrics in this function
	return m.customMetrics[key]
}

func (m Metrics) UpsertCustomMetric(key string, metric common.Metric) {
	m.customMetrics[key] = metric
}

func (m Metrics) UpdatePersisted(kind string) {
	if _, ok := m.updatesPersisted[kind]; ok {
		m.updatesPersisted[kind].Inc()
	}
}

func (m Metrics) PodProcessed() {
	m.podsProcessed.Inc()
}

func (m Metrics) UpdatesReapplied() {
	m.updatesReapplied.Inc()
}

func (m Metrics) PodResourceVersionRepeated() {
	m.podResourceVersionRepeated.Inc()
}

func (m Metrics) PodChanged(significant bool) {
	m.podChange[significant].Inc()
}

func (m Metrics) WorkflowResourceVersionRepeated() {
	m.workflowResourceVersionRepeated.Inc()
}
