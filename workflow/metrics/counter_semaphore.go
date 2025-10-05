package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func addSemaphoreTakenCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentSemaphoreTakenTotal)
}

func addSemaphoreReleasedCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentSemaphoreReleasedTotal)
}
