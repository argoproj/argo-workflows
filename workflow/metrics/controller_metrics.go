package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
)

// legacyWorkflowCollector collects metrics about all workflows in the cluster
type controllerCollector struct {
	store                   util.WorkflowLister
	gauges                  []prometheus.Metric
	lastSyncResourceVersion string
}

// Describe implements the prometheus.Collector interface
func (wc *controllerCollector) Describe(ch chan<- *prometheus.Desc) {
	if wc.lastSyncResourceVersion != wc.store.LastSyncResourceVersion() {
		workflows, err := wc.store.List()
		if err != nil {
			return
		}
		wc.loadWorkflowStatuses(workflows)
		wc.lastSyncResourceVersion = wc.store.LastSyncResourceVersion()
	}
	for _, metric := range wc.gauges {
		ch <- metric.Desc()
	}
}

// Collect implements the prometheus.Collector interface
func (wc *controllerCollector) Collect(ch chan<- prometheus.Metric) {
	if wc.lastSyncResourceVersion != wc.store.LastSyncResourceVersion() {
		workflows, err := wc.store.List()
		if err != nil {
			return
		}
		wc.loadWorkflowStatuses(workflows)
		wc.lastSyncResourceVersion = wc.store.LastSyncResourceVersion()
	}
	for _, metric := range wc.gauges {
		ch <- metric
	}
}

func (wc *controllerCollector) loadWorkflowStatuses(wfs []*wfv1.Workflow) {
	if len(wfs) == 0 {
		return
	}

	getOptsByPahse := func(phase wfv1.NodePhase) prometheus.GaugeOpts {
		return prometheus.GaugeOpts{
			Namespace:   argoNamespace,
			Subsystem:   workflowsSubsystem,
			Name:        "count",
			Help:        "Number of Workflows currently accessible by the controller by status",
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
		if _, ok := gauges[wf.Status.Phase]; ok {
			gauges[wf.Status.Phase].Inc()
		}
	}

	var out []prometheus.Metric
	for _, gauge := range gauges {
		out = append(out, gauge)
	}
	wc.gauges = out
}
