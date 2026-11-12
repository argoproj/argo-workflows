package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// LockGaugeSample is a point-in-time snapshot of a single synchronization lock's occupancy.
type LockGaugeSample struct {
	Type      string // "mutex" or "semaphore"
	Storage   string // "configmap" or "database"
	Name      string // the lock's resource name
	Namespace string // the lock's namespace
	Held      int64  // holders currently holding the lock
	Pending   int64  // requests currently waiting to acquire
}

// LocksCallback is the function prototype that provides the lock gauges with the current
// per-lock held/pending snapshot. It is invoked at metric scrape time.
type LocksCallback func(ctx context.Context) []LockGaugeSample

type locksGauge struct {
	callback LocksCallback
	observe  func(ctx context.Context, o metric.Observer, val int64, lockType, storage, name, namespace string)
	value    func(LockGaugeSample) int64
}

// addLocksGauges creates the lock gauge instruments. Their observing callback is registered later,
// once the sync Manager exists, via RegisterLockGauges - so nothing reads a shared manager pointer
// on the scrape path.
func addLocksGauges(_ context.Context, m *Metrics) error {
	if err := m.CreateBuiltinInstrument(telemetry.InstrumentLocksHeld); err != nil {
		return err
	}
	return m.CreateBuiltinInstrument(telemetry.InstrumentLocksPending)
}

// RegisterLockGauges wires the locks_held / locks_pending gauges to their data source. It is called
// by the sync Manager (from WithMetrics) once it exists, binding the observable callbacks directly
// to the live snapshot function. A no-op if cb is nil.
func (m *Metrics) RegisterLockGauges(cb LocksCallback) error {
	if cb == nil {
		return nil
	}
	held := &locksGauge{
		callback: cb,
		observe:  m.ObserveLocksHeld,
		value:    func(s LockGaugeSample) int64 { return s.Held },
	}
	if inst := m.GetInstrument(telemetry.InstrumentLocksHeld.Name()); inst != nil {
		if err := inst.RegisterCallback(m.Metrics, held.update); err != nil {
			return err
		}
	}

	pending := &locksGauge{
		callback: cb,
		observe:  m.ObserveLocksPending,
		value:    func(s LockGaugeSample) int64 { return s.Pending },
	}
	if inst := m.GetInstrument(telemetry.InstrumentLocksPending.Name()); inst != nil {
		return inst.RegisterCallback(m.Metrics, pending.update)
	}
	return nil
}

func (g *locksGauge) update(ctx context.Context, o metric.Observer) error {
	for _, s := range g.callback(ctx) {
		g.observe(ctx, o, g.value(s), s.Type, s.Storage, s.Name, s.Namespace)
	}
	return nil
}
