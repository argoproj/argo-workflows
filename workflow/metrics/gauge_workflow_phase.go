package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

// WorkflowPhaseCallback is the function prototype to provide this gauge with the phase of the pods
type WorkflowPhaseCallback func() map[string]int64

type workflowPhaseGauge struct {
	callback WorkflowPhaseCallback
	gauge    *instrument
}

func addWorkflowPhaseGauge(_ context.Context, m *Metrics) error {
	const nameWorkflowPhaseGauge = `gauge`
	err := m.createInstrument(int64ObservableGauge,
		nameWorkflowPhaseGauge,
		"number of Workflows currently accessible by the controller by status",
		"{workflow}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	if m.callbacks.WorkflowPhase != nil {
		wfpGauge := workflowPhaseGauge{
			callback: m.callbacks.WorkflowPhase,
			gauge:    m.allInstruments[nameWorkflowPhaseGauge],
		}
		return m.allInstruments[nameWorkflowPhaseGauge].registerCallback(m, wfpGauge.update)
	}
	return nil
	// TODO init all phases?
}

func (p *workflowPhaseGauge) update(_ context.Context, o metric.Observer) error {
	phases := p.callback()
	for phase, val := range phases {
		p.gauge.observeInt(o, val, instAttribs{{name: labelWorkflowStatus, value: phase}})
	}
	return nil
}
