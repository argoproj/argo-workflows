package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

// PoolGaugeSample is a point-in-time snapshot of a single hot worker pool's queue and workers.
type PoolGaugeSample struct {
	Namespace string // the pool's namespace
	Name      string // the pool's name
	Hash      string // the pool's hash
	Backlog   int64  // tasks waiting for a worker
	Claimed   int64  // tasks claimed and not yet completed
	Ready     int64  // worker pods ready and not being deleted
}

// PoolsCallback is the function prototype that provides the pool gauges with the current
// per-pool snapshot. It is invoked at metric scrape time.
type PoolsCallback func(ctx context.Context) []PoolGaugeSample

// addPoolGauges creates the pool gauge instruments. Their observing callback is registered later,
// once the controller's informers exist, via RegisterPoolGauges.
func addPoolGauges(_ context.Context, m *Metrics) error {
	for _, inst := range []telemetry.BuiltinInstrument{
		telemetry.InstrumentPoolBacklog,
		telemetry.InstrumentPoolClaimedTasks,
		telemetry.InstrumentPoolReadyPods,
	} {
		if err := m.CreateBuiltinInstrument(inst); err != nil {
			return err
		}
	}
	return nil
}

// RegisterPoolGauges wires the pool_backlog / pool_claimed_tasks / pool_ready_pods gauges to their
// data source. It is called by the controller once its informers exist, binding one observable
// callback for all three to the live snapshot function, so that a scrape reads the caches once and
// the three gauges agree. A no-op if cb is nil.
func (m *Metrics) RegisterPoolGauges(cb PoolsCallback) error {
	if cb == nil {
		return nil
	}
	var instruments []*telemetry.Instrument
	for _, inst := range []telemetry.BuiltinInstrument{
		telemetry.InstrumentPoolBacklog,
		telemetry.InstrumentPoolClaimedTasks,
		telemetry.InstrumentPoolReadyPods,
	} {
		if i := m.GetInstrument(inst.Name()); i != nil {
			instruments = append(instruments, i)
		}
	}
	if len(instruments) == 0 {
		return nil
	}
	return m.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		for _, s := range cb(ctx) {
			m.ObservePoolBacklog(ctx, o, s.Backlog, s.Namespace, s.Name, s.Hash)
			m.ObservePoolClaimedTasks(ctx, o, s.Claimed, s.Namespace, s.Name, s.Hash)
			m.ObservePoolReadyPods(ctx, o, s.Ready, s.Namespace, s.Name, s.Hash)
		}
		return nil
	}, instruments...)
}
