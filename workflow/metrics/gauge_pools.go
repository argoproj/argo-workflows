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

type poolsGauge struct {
	callback PoolsCallback
	observe  func(ctx context.Context, o metric.Observer, val int64, namespace, name, hash string)
	value    func(PoolGaugeSample) int64
}

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
// data source. It is called by the controller once its informers exist, binding the observable
// callbacks directly to the live snapshot function. A no-op if cb is nil.
func (m *Metrics) RegisterPoolGauges(cb PoolsCallback) error {
	if cb == nil {
		return nil
	}
	for name, gauge := range map[string]*poolsGauge{
		telemetry.InstrumentPoolBacklog.Name(): {
			callback: cb,
			observe:  m.ObservePoolBacklog,
			value:    func(s PoolGaugeSample) int64 { return s.Backlog },
		},
		telemetry.InstrumentPoolClaimedTasks.Name(): {
			callback: cb,
			observe:  m.ObservePoolClaimedTasks,
			value:    func(s PoolGaugeSample) int64 { return s.Claimed },
		},
		telemetry.InstrumentPoolReadyPods.Name(): {
			callback: cb,
			observe:  m.ObservePoolReadyPods,
			value:    func(s PoolGaugeSample) int64 { return s.Ready },
		},
	} {
		if inst := m.GetInstrument(name); inst != nil {
			if err := inst.RegisterCallback(m.Metrics, gauge.update); err != nil {
				return err
			}
		}
	}
	return nil
}

func (g *poolsGauge) update(ctx context.Context, o metric.Observer) error {
	for _, s := range g.callback(ctx) {
		g.observe(ctx, o, g.value(s), s.Namespace, s.Name, s.Hash)
	}
	return nil
}
