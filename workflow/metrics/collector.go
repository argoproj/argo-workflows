package metrics

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/tools/cache"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

var (
	descWorkflowDefaultLabels                 = []string{"namespace", "name", "entrypoint"}
	descWorkflowStepDefaultLabels             = []string{"namespace", "name", "step_name"}
	descWorkflowNodeCustomMetricDefaultLabels = append(descWorkflowStepDefaultLabels, "metric_name")

	descWorkflowInfo = prometheus.NewDesc(
		"argo_workflow_info",
		"Information about workflow.",
		append(descWorkflowDefaultLabels, "service_account_name", "templates"),
		nil,
	)
	descWorkflowStartedAt = prometheus.NewDesc(
		"argo_workflow_start_time",
		"Start time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowFinishedAt = prometheus.NewDesc(
		"argo_workflow_completion_time",
		"Completion time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowCreated = prometheus.NewDesc(
		"argo_workflow_created_time",
		"Creation time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowStatusPhase = prometheus.NewDesc(
		"argo_workflow_status_phase",
		"The workflow current phase.",
		append(descWorkflowDefaultLabels, "phase"),
		nil,
	)
	descWorkflowNodeStartedAt = prometheus.NewDesc(
		"argo_workflow_step_start_time",
		"Start time in unix timestamp for a workflow step.",
		descWorkflowStepDefaultLabels,
		nil,
	)
	descWorkflowNodeFinishedAt = prometheus.NewDesc(
		"argo_workflow_step_completion_time",
		"Completion time in unix timestamp for a workflow step.",
		descWorkflowStepDefaultLabels,
		nil,
	)
	descWorkflowNodeStatusPhase = prometheus.NewDesc(
		"argo_workflow_step_status_phase",
		"The workflow step current phase.",
		append(descWorkflowStepDefaultLabels, "phase"),
		nil,
	)
)

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// workflowCollector collects metrics about all workflows in the cluster
type workflowCollector struct {
	store util.WorkflowLister
}

// NewWorkflowRegistry creates a new prometheus registry that collects workflows
func NewWorkflowRegistry(informer cache.SharedIndexInformer) *prometheus.Registry {
	workflowLister := util.NewWorkflowLister(informer)
	registry := prometheus.NewRegistry()
	registry.MustRegister(&workflowCollector{store: workflowLister})
	return registry
}

// NewTelemetryRegistry creates a new prometheus registry that collects telemetry
func NewTelemetryRegistry() *prometheus.Registry {
	registry := prometheus.NewRegistry()
	registry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	registry.MustRegister(prometheus.NewGoCollector())
	return registry
}

// Describe implements the prometheus.Collector interface
func (wc *workflowCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descWorkflowInfo
	ch <- descWorkflowStartedAt
	ch <- descWorkflowFinishedAt
	ch <- descWorkflowCreated
	ch <- descWorkflowStatusPhase
}

// Collect implements the prometheus.Collector interface
func (wc *workflowCollector) Collect(ch chan<- prometheus.Metric) {
	workflows, err := wc.store.List()
	if err != nil {
		return
	}
	for _, wf := range workflows {
		wc.collectWorkflow(ch, *wf)
	}
}

func (wc *workflowCollector) collectWorkflow(ch chan<- prometheus.Metric, wf wfv1.Workflow) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{wf.Namespace, wf.Name, wf.Spec.Entrypoint}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}
	joinTemplates := func(spec []wfv1.Template) string {
		var templates []string
		for _, t := range spec {
			templates = append(templates, t.Name)
		}
		return strings.Join(templates, ",")
	}

	addGauge(descWorkflowInfo, 1, wf.Spec.ServiceAccountName, joinTemplates(wf.Spec.Templates))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodePending || wf.Status.Phase == ""), string(wfv1.NodePending))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodeRunning), string(wfv1.NodeRunning))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodeSucceeded), string(wfv1.NodeSucceeded))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodeSkipped), string(wfv1.NodeSkipped))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodeFailed), string(wfv1.NodeFailed))
	addGauge(descWorkflowStatusPhase, boolFloat64(wf.Status.Phase == wfv1.NodeError), string(wfv1.NodeError))

	if !wf.CreationTimestamp.IsZero() {
		addGauge(descWorkflowCreated, float64(wf.CreationTimestamp.Unix()))
	}

	if !wf.Status.StartedAt.IsZero() {
		addGauge(descWorkflowStartedAt, float64(wf.Status.StartedAt.Unix()))
	}

	if !wf.Status.FinishedAt.IsZero() {
		addGauge(descWorkflowFinishedAt, float64(wf.Status.FinishedAt.Unix()))
	}

	// Collect Node metrics
	for _, node := range wf.Status.Nodes {
		wc.collectWorkflowNode(ch, node, wf.Name, wf.Namespace)
	}
}

func (wc *workflowCollector) collectWorkflowNode(ch chan<- prometheus.Metric, node wfv1.NodeStatus, wfName, wfNamespace string) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{wfNamespace, wfName, node.Name}, lv...)
		ch <- prometheus.MustNewConstMetric(desc, t, v, lv...)
	}
	addGauge := func(desc *prometheus.Desc, v float64, lv ...string) {
		addConstMetric(desc, prometheus.GaugeValue, v, lv...)
	}

	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodePending || node.Phase == ""), string(wfv1.NodePending))
	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodeRunning), string(wfv1.NodeRunning))
	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodeSucceeded), string(wfv1.NodeSucceeded))
	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodeSkipped), string(wfv1.NodeSkipped))
	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodeFailed), string(wfv1.NodeFailed))
	addGauge(descWorkflowNodeStatusPhase, boolFloat64(node.Phase == wfv1.NodeError), string(wfv1.NodeError))

	if !node.StartedAt.IsZero() {
		addGauge(descWorkflowNodeStartedAt, float64(node.StartedAt.Unix()))
	}

	if !node.FinishedAt.IsZero() {
		addGauge(descWorkflowNodeFinishedAt, float64(node.FinishedAt.Unix()))
	}

	if node.Outputs != nil {
		for _, param := range node.Outputs.Parameters {
			if param.EmitMetric {
				metricDesc := prometheus.NewDesc(
					"argo_workflow_" + param.Name,
					fmt.Sprintf("Custom metric '%s' from Workflow '%s'", param.Name, wfName),
					descWorkflowStepDefaultLabels,
					nil,
				)

				parsedValue, err := strconv.ParseFloat(*param.Value, 64)
				if err == nil {
					addGauge(metricDesc, parsedValue)
				} else {
					log.Infof("Not able to add value as metric")
				}
			}
		}
	}
}
