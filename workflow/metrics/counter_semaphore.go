package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

func addSemaphoreCounter(_ context.Context, m *Metrics) error {
    return m.CreateBuiltinInstrument(telemetry.InstrumentSemaphoreTotal)
}
