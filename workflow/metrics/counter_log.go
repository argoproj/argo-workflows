package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/telemetry"

	log "github.com/sirupsen/logrus"
)

type logMetric struct {
	counter *telemetry.Instrument
}

func addLogCounter(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentLogMessages)
	name := telemetry.InstrumentLogMessages.Name()
	lm := logMetric{
		counter: m.GetInstrument(name),
	}
	log.AddHook(lm)
	for _, level := range lm.Levels() {
		m.AddInt(ctx, name, 0, telemetry.InstAttribs{
			{Name: telemetry.AttribLogLevel, Value: level.String()},
		})
	}

	return err
}

func (m logMetric) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.WarnLevel, log.ErrorLevel}
}

func (m logMetric) Fire(entry *log.Entry) error {
	(*m.counter).AddInt(entry.Context, 1, telemetry.InstAttribs{
		{Name: telemetry.AttribLogLevel, Value: entry.Level.String()},
	})
	return nil
}
