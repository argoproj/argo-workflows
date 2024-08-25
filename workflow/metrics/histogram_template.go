package metrics

import (
	"context"
	"time"
)

const (
	nameWorkflowTemplateRuntime = `workflowtemplate_runtime`
)

func addWorkflowTemplateHistogram(_ context.Context, m *Metrics) error {
	return m.createInstrument(float64Histogram,
		nameWorkflowTemplateRuntime,
		"Duration of workflow template runs run through workflowTemplateRefs",
		"s",
		withAsBuiltIn(),
	)
}

func (m *Metrics) RecordWorkflowTemplateTime(ctx context.Context, duration time.Duration, name, namespace string, cluster bool) {
	m.record(ctx, nameWorkflowTemplateRuntime, duration.Seconds(), templateLabels(name, namespace, cluster))
}
