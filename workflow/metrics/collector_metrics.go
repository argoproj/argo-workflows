package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

// legacyWorkflowCollector collects metrics about all workflows in the cluster
type controllerCollector struct {
	store util.WorkflowLister
}

// Describe implements the prometheus.Collector interface
func (wc *controllerCollector) Describe(ch chan<- *prometheus.Desc) {
	workflows, err := wc.store.List()
	if err != nil {
		return
	}
	for _, metric := range wc.collectWorkflowStatuses(workflows) {
		ch <- metric.Desc()
	}
}

// Collect implements the prometheus.Collector interface
func (wc *controllerCollector) Collect(ch chan<- prometheus.Metric) {
	workflows, err := wc.store.List()
	if err != nil {
		return
	}
	for _, metric := range wc.collectWorkflowStatuses(workflows) {
		ch <- metric
	}
}

func (wc *controllerCollector) collectWorkflowStatuses(wfs []*wfv1.Workflow) []prometheus.Metric {
	if len(wfs) == 0 {
		return nil
	}

	getOptsByPahse := func(phase wfv1.NodePhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "workflows_by_status_count",
			Help:        "Number of Workflows currently accesible by the controller by status",
			ConstLabels: map[string]string{"status": string(phase)},
		}
	}
	gauges := map[wfv1.NodePhase]prometheus.Gauge{
		wfv1.NodePending:   prometheus.NewGauge(getOptsByPahse(wfv1.NodePending)),
		wfv1.NodeRunning:   prometheus.NewGauge(getOptsByPahse(wfv1.NodeRunning)),
		wfv1.NodeSucceeded: prometheus.NewGauge(getOptsByPahse(wfv1.NodeSucceeded)),
		wfv1.NodeSkipped:   prometheus.NewGauge(getOptsByPahse(wfv1.NodeSkipped)),
		wfv1.NodeFailed:    prometheus.NewGauge(getOptsByPahse(wfv1.NodeFailed)),
		wfv1.NodeError:     prometheus.NewGauge(getOptsByPahse(wfv1.NodeError)),
	}

	for _, wf := range wfs {
		gauges[wf.Status.Phase].Inc()
	}

	var out []prometheus.Metric
	for _, gauge := range gauges {
		out = append(out, gauge)
	}
	return out
}
