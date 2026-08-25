package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"
)

func addLocksTakenCounter(_ context.Context, m *Metrics) error {
	return m.CreateBuiltinInstrument(telemetry.InstrumentLocksTakenTotal)
}

// RecordLockTaken increments the counter of synchronization locks acquired.
func (m *Metrics) RecordLockTaken(ctx context.Context, lockType, storage, name, namespace string) {
	if m == nil || m.Metrics == nil {
		return
	}
	m.AddLocksTakenTotal(ctx, 1, lockType, storage, name, namespace)
}
