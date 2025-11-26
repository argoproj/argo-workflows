package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addMutexTakenCounter(_ context.Context, m *Metrics) error {
    return m.CreateBuiltinInstrument(telemetry.InstrumentMutexTakenTotal)
}

func addMutexReleasedCounter(_ context.Context, m *Metrics) error {
    return m.CreateBuiltinInstrument(telemetry.InstrumentMutexReleasedTotal)
}
