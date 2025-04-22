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
	const nameLeader = `is_leader`
	err := m.CreateInstrument(telemetry.Int64ObservableGauge,
		nameLeader,
		"Emits 1 if leader, 0 otherwise. Always 1 if leader election is disabled.",
		"{leader}",
		telemetry.WithAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	if m.callbacks.IsLeader == nil {
		return nil
	}
	lGauge := leaderGauge{
		callback: m.callbacks.IsLeader,
		gauge:    m.GetInstrument(nameLeader),
	}
	return lGauge.gauge.RegisterCallback(m.Metrics, lGauge.update)
}

func (l *leaderGauge) update(_ context.Context, o metric.Observer) error {
	var val int64 = 0
	if l.callback() {
		val = 1
	}
	l.gauge.ObserveInt(o, val, telemetry.InstAttribs{})
	return nil
}
