package metrics

import (
	"os"
	"strings"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

var (
	descWorkflowDefaultLabels = []string{"namespace", "name"}

	descWorkflowInfo = prometheus.NewDesc(
		"kube_wf_info",
		"Information about workflow.",
		append(descWorkflowDefaultLabels, "entrypoint", "service_account_name", "templates"),
		nil,
	)
	descWorkflowStartedAt = prometheus.NewDesc(
		"kube_wf_start_time",
		"Start time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowFinishedAt = prometheus.NewDesc(
		"kube_wf_completion_time",
		"Completion time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowCreated = prometheus.NewDesc(
		"kube_wf_created",
		"Creation time in unix timestamp for a workflow.",
		descWorkflowDefaultLabels,
		nil,
	)
	descWorkflowStatusPhase = prometheus.NewDesc(
		"kube_wf_status_phase",
		"The workflow current phase.",
		append(descWorkflowDefaultLabels, "phase"),
		nil,
	)
)

func boolFloat64(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

type workflowLister func() ([]wfv1.Workflow, error)

func (l workflowLister) List() ([]wfv1.Workflow, error) {
	return l()
}

type wfStore interface {
	List() (workflows []wfv1.Workflow, err error)
}

// workflowCollector collects metrics about all workflows in the cluster
type workflowCollector struct {
	store wfStore
}

// NewWorkflowRegistry creates a new prometheus registry that collects workflows
func NewWorkflowRegistry(informer cache.SharedIndexInformer) *prometheus.Registry {
	workflowLister := workflowLister(func() (workflows []wfv1.Workflow, err error) {
		for _, m := range informer.GetStore().List() {
			var wf wfv1.Workflow
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(m.(*unstructured.Unstructured).Object, &wf)
			if err != nil {
				return nil, err
			}
			workflows = append(workflows, wf)
		}
		return workflows, nil
	})
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
		wc.collectWorkflow(ch, wf)
	}
}

func (wc *workflowCollector) collectWorkflow(ch chan<- prometheus.Metric, wf wfv1.Workflow) {
	addConstMetric := func(desc *prometheus.Desc, t prometheus.ValueType, v float64, lv ...string) {
		lv = append([]string{wf.Namespace, wf.Name}, lv...)
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

	addGauge(descWorkflowInfo, 1, wf.Spec.Entrypoint, wf.Spec.ServiceAccountName, joinTemplates(wf.Spec.Templates))

	if phase := wf.Status.Phase; phase != "" {
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeRunning), string(wfv1.NodeRunning))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeSucceeded), string(wfv1.NodeSucceeded))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeSkipped), string(wfv1.NodeSkipped))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeFailed), string(wfv1.NodeFailed))
		addGauge(descWorkflowStatusPhase, boolFloat64(phase == wfv1.NodeError), string(wfv1.NodeError))
	}

	if !wf.CreationTimestamp.IsZero() {
		addGauge(descWorkflowCreated, float64(wf.CreationTimestamp.Unix()))
	}

	if !wf.Status.StartedAt.IsZero() {
		addGauge(descWorkflowStartedAt, float64(wf.Status.StartedAt.Unix()))
	}

	if !wf.Status.FinishedAt.IsZero() {
		addGauge(descWorkflowFinishedAt, float64(wf.Status.FinishedAt.Unix()))
	}

}
