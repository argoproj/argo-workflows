package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	"go.opentelemetry.io/otel/metric"
)

type IsLeaderCallback func() bool

type leaderGauge struct {
	callback IsLeaderCallback
	gauge    *telemetry.Instrument
}

func addIsLeader(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentIsLeader)
	if err != nil {
		return err
	}
	if m.callbacks.IsLeader == nil {
		return nil
	}
	name := telemetry.InstrumentIsLeader.Name()
	lGauge := leaderGauge{
		callback: m.callbacks.IsLeader,
		gauge:    m.GetInstrument(name),
	}
	return lGauge.gauge.RegisterCallback(m.Metrics, lGauge.update)
}

func (l *leaderGauge) update(ctx context.Context, o metric.Observer) error {
	var val int64 = 0
	if l.callback() {
		val = 1
	}
	l.gauge.ObserveInt(ctx, o, val, telemetry.InstAttribs{})
	return nil
}
