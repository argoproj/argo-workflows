package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// WorkflowConditionCallback is the function prototype to provide this gauge with the condition of the workflows
type WorkflowConditionCallback func() map[wfv1.Condition]int64

type workflowConditionGauge struct {
	callback WorkflowConditionCallback
	gauge    *instrument
}

func addWorkflowConditionGauge(_ context.Context, m *Metrics) error {
	const nameWorkflowCondition = `workflow_condition`
	err := m.createInstrument(int64ObservableGauge,
		nameWorkflowCondition,
		"Workflow condition.",
		"{unit}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	if m.callbacks.WorkflowCondition != nil {
		wfcGauge := workflowConditionGauge{
			callback: m.callbacks.WorkflowCondition,
			gauge:    m.allInstruments[nameWorkflowCondition],
		}
		return m.allInstruments[nameWorkflowCondition].registerCallback(m, wfcGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (c *workflowConditionGauge) update(_ context.Context, o metric.Observer) error {
	conditions := c.callback()
	for condition, val := range conditions {
		c.gauge.observeInt(o, val, instAttribs{
			{name: labelWorkflowType, value: string(condition.Type)},
			{name: labelWorkflowStatus, value: string(condition.Status)},
		})
	}
	return nil
}
