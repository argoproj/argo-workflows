package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v4/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

type IsLeaderCallback func() bool

type leaderGauge struct {
	callback IsLeaderCallback
	observe  func(ctx context.Context, o metric.Observer, val int64)
}

func addIsLeader(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentIsLeader)
	if err != nil {
		return err
	}
	if m.callbacks.IsLeader == nil {
		return nil
	}
	inst := m.GetInstrument(telemetry.InstrumentIsLeader.Name())
	lGauge := leaderGauge{
		callback: m.callbacks.IsLeader,
		observe:  m.ObserveIsLeader,
	}
	return inst.RegisterCallback(m.Metrics, lGauge.update)
}

func (l *leaderGauge) update(ctx context.Context, o metric.Observer) error {
	var val int64 = 0
	if l.callback() {
		val = 1
	}
	l.observe(ctx, o, val)
	return nil
}
