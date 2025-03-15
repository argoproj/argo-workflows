package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

// WorkflowConditionCallback is the function prototype to provide this gauge with the condition of the workflows
type WorkflowConditionCallback func() map[wfv1.Condition]int64

type workflowConditionGauge struct {
	callback WorkflowConditionCallback
	gauge    *telemetry.Instrument
}

func addWorkflowConditionGauge(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowCondition)
	if err != nil {
		return err
	}

	if m.callbacks.WorkflowCondition != nil {
		wfcGauge := workflowConditionGauge{
			callback: m.callbacks.WorkflowCondition,
			gauge:    m.GetInstrument(telemetry.InstrumentWorkflowCondition.Name()),
		}
		return wfcGauge.gauge.RegisterCallback(m.Metrics, wfcGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (c *workflowConditionGauge) update(_ context.Context, o metric.Observer) error {
	conditions := c.callback()
	for condition, val := range conditions {
		c.gauge.ObserveInt(o, val, telemetry.InstAttribs{
			{Name: telemetry.AttribWorkflowType, Value: string(condition.Type)},
			{Name: telemetry.AttribWorkflowStatus, Value: string(condition.Status)},
		})
	}
	return nil
}
