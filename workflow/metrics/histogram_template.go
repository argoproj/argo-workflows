package metrics

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

const (
	nameWorkflowTemplateRuntime = `workflowtemplate_runtime`
)

func addWorkflowTemplateHistogram(_ context.Context, m *Metrics) error {
	return m.CreateInstrument(telemetry.Float64Histogram,
		nameWorkflowTemplateRuntime,
		"Duration of workflow template runs run through workflowTemplateRefs",
		"s",
		telemetry.WithAsBuiltIn(),
	)
}

func (m *Metrics) RecordWorkflowTemplateTime(ctx context.Context, duration time.Duration, name, namespace string, cluster bool) {
	m.Record(ctx, nameWorkflowTemplateRuntime, duration.Seconds(), templateAttribs(name, namespace, cluster))
}
