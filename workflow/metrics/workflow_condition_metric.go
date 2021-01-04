package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	WorkflowConditionMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: argoNamespace,
			Subsystem: workflowsSubsystem,
			Name:      "workflow_condition",
			Help:      "Workflow condition. https://argoproj.github.io/argo/metrics/#workflow_condition",
		},
		[]string{"type", "status"},
	)
)
