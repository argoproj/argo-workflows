package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type LeaderStateCallback func() bool

type leaderGauge struct {
	callback LeaderStateCallback
	gauge    *instrument
}

func addIsLeader(ctx context.Context, m *Metrics) error {
	const nameLeader = `leader`
	err := m.createInstrument(int64ObservableGauge,
		nameLeader,
		"Emits 1 if this is the leader when leader elections are enabled, or 0 otherwise. Always 1 when leader elections are disabled.",
		"{leader}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	if m.callbacks.LeaderState != nil {
		lGauge := leaderGauge{
			callback: m.callbacks.LeaderState,
			gauge:    m.allInstruments[nameLeader],
		}
		return m.allInstruments[nameLeader].registerCallback(m, lGauge.update)
	}
	return nil
}

func (l *leaderGauge) update(_ context.Context, o metric.Observer) error {
	var val int64 = 0
	if l.callback() {
		val = 1
	}
	l.gauge.observeInt(o, val, instAttribs{})
	return nil
}
