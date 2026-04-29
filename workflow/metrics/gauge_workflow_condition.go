package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

// WorkflowConditionCallback is the function prototype to provide this gauge with the condition of the workflows
type WorkflowConditionCallback func(ctx context.Context) map[wfv1.Condition]int64

type workflowConditionGauge struct {
	callback WorkflowConditionCallback
	observe  func(ctx context.Context, o metric.Observer, val int64, conditionType string, conditionStatus string)
}

func addWorkflowConditionGauge(_ context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowCondition)
	if err != nil {
		return err
	}

	if m.callbacks.WorkflowCondition != nil {
		inst := m.GetInstrument(telemetry.InstrumentWorkflowCondition.Name())
		wfcGauge := workflowConditionGauge{
			callback: m.callbacks.WorkflowCondition,
			observe:  m.ObserveWorkflowCondition,
		}
		return inst.RegisterCallback(m.Metrics, wfcGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (c *workflowConditionGauge) update(ctx context.Context, o metric.Observer) error {
	conditions := c.callback(ctx)
	for condition, val := range conditions {
		c.observe(ctx, o, val, string(condition.Type), string(condition.Status))
	}
	return nil
}
