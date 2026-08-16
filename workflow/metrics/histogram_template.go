package metrics

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func addWorkflowTemplateHistogram(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowtemplateRuntime)
}

func (m *Metrics) RecordWorkflowTemplateTime(ctx context.Context, duration time.Duration, name, namespace string, cluster bool) {
	m.RecordWorkflowtemplateRuntime(ctx, duration.Seconds(), name, namespace, cluster)
}
