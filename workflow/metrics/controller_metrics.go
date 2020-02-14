package metrics

import (
	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo/workflow/util"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	descControllerWorkflowsGauge = prometheus.NewDesc(
		"argo_workflows_total",
		"Total number of Workflows by phase",
		[]string{"phase"},
		nil,
		)

	descControllerWorkflowTimeToStartGauge = prometheus.NewDesc(
		"argo_workflows_time_to_start_seconds",
		"Average time in seconds that it takes a workflow to start once it is created",
		[]string{},
		nil,
	)
)

type controllerMetricCollector struct {
	store util.WorkflowLister
}

// Describe implements the prometheus.Collector interface
func (cmc *controllerMetricCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- descControllerWorkflowsGauge
}

// Collect implements the prometheus.Collector interface
func (cmc *controllerMetricCollector) Collect(ch chan<- prometheus.Metric) {
	workflows, err := cmc.store.List()
	if err != nil {
		return
	}

	cmc.collectTotalWorkflows(ch, workflows)
}

func (cmc *controllerMetricCollector) collectTotalWorkflows(ch chan<- prometheus.Metric, workflows []*v1alpha1.Workflow) {
	workflowPhases := map[v1alpha1.NodePhase]float64{
		v1alpha1.NodePending: 0,
		v1alpha1.NodeRunning: 0,
		v1alpha1.NodeSucceeded: 0,
		v1alpha1.NodeSkipped: 0,
		v1alpha1.NodeFailed: 0,
		v1alpha1.NodeError: 0,
	}
	averageTimeToStart := 0.0

	for i, wf := range workflows {
		if !wf.Status.StartedAt.IsZero() {
			timeToStart := wf.Status.StartedAt.Time.Sub(wf.ObjectMeta.CreationTimestamp.Time).Seconds()
			averageTimeToStart = getAverageFromStream(averageTimeToStart, timeToStart, i)
		}

		if wf.Status.Phase != "" {
			if _, ok := workflowPhases[wf.Status.Phase]; ok {
				workflowPhases[wf.Status.Phase]++
			}
		}
	}

	for key, value := range workflowPhases {
		ch <- prometheus.MustNewConstMetric(descControllerWorkflowsGauge, prometheus.GaugeValue, value, string(key))
	}
	ch <- prometheus.MustNewConstMetric(descControllerWorkflowTimeToStartGauge, prometheus.GaugeValue, averageTimeToStart)
}

func getAverageFromStream(prevAverage, newItem float64, currentIndex int) float64 {
	return ((prevAverage * float64(currentIndex)) + newItem) / float64(currentIndex + 1)
}
