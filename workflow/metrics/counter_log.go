package metrics

import (
	"context"

	log "github.com/sirupsen/logrus"
)

type logMetric struct {
	counter *instrument
}

func addLogCounter(ctx context.Context, m *Metrics) error {
	const nameLogMessages = `log_messages`
	err := m.createInstrument(int64Counter,
		nameLogMessages,
		"Total number of log messages.",
		"{message}",
		withAsBuiltIn(),
	)
	lm := logMetric{
		counter: m.allInstruments[nameLogMessages],
	}
	log.AddHook(lm)
	for _, level := range lm.Levels() {
		m.addInt(ctx, nameLogMessages, 0, instAttribs{
			{name: labelLogLevel, value: level.String()},
		})
	}

	return err
}

func (m logMetric) Levels() []log.Level {
	return []log.Level{log.InfoLevel, log.WarnLevel, log.ErrorLevel}
}

func (m logMetric) Fire(entry *log.Entry) error {
	(*m.counter).addInt(entry.Context, 1, instAttribs{
		{name: labelLogLevel, value: entry.Level.String()},
	})
	return nil
}
