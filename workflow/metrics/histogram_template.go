package metrics

import (
	"context"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addWorkflowTemplateHistogram(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentWorkflowtemplateRuntime)
}

func (m *Metrics) RecordWorkflowTemplateTime(ctx context.Context, duration time.Duration, name, namespace string, cluster bool) {
	m.Record(ctx, telemetry.InstrumentWorkflowtemplateRuntime.Name(), duration.Seconds(), templateAttribs(name, namespace, cluster))
}
