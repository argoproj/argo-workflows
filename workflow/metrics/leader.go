package metrics

import (
	"context"

	"go.opentelemetry.io/otel/metric"
)

type IsLeaderCallback func() bool

type leaderGauge struct {
	callback IsLeaderCallback
	gauge    *instrument
}

func addIsLeader(ctx context.Context, m *Metrics) error {
	const nameLeader = `is_leader`
	err := m.createInstrument(int64ObservableGauge,
		nameLeader,
		"Emits 1 if leader, 0 otherwise. Always 1 if leader election is disabled.",
		"{leader}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}
	if m.callbacks.IsLeader == nil {
		return nil
	}
	lGauge := leaderGauge{
		callback: m.callbacks.IsLeader,
		gauge:    m.allInstruments[nameLeader],
	}
	return m.allInstruments[nameLeader].registerCallback(m, lGauge.update)
}

func (l *leaderGauge) update(_ context.Context, o metric.Observer) error {
	var val int64 = 0
	if l.callback() {
		val = 1
	}
	l.gauge.observeInt(o, val, instAttribs{})
	return nil
}
