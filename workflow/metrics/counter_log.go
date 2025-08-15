package metrics

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

var globalLogCounter unsafe.Pointer // *telemetry.Instrument

// init registers the metrics hook at package initialization time
func init() {
	// Register the global hook immediately
	logging.AddGlobalHook(&globalLogMetric{})
}

type globalLogMetric struct{}

func (g *globalLogMetric) Levels() []logging.Level {
	return []logging.Level{logging.Debug, logging.Info, logging.Warn, logging.Error}
}

func (g *globalLogMetric) Fire(ctx context.Context, level logging.Level, _ string, _ logging.Fields) {
	// Get the counter atomically
	counterPtr := (*telemetry.Instrument)(atomic.LoadPointer(&globalLogCounter))

	// Only fire if we have a real counter
	if counterPtr != nil {
		(*counterPtr).AddInt(ctx, 1, telemetry.InstAttribs{
			{Name: telemetry.AttribLogLevel, Value: logLevelName(level)},
		})
	}
}

func logLevelName(level logging.Level) string {
	switch level {
	case logging.Warn:
		return "warning"
	default:
		return string(level)
	}
}

func addLogCounter(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(telemetry.InstrumentLogMessages)
	name := telemetry.InstrumentLogMessages.Name()
	counter := m.GetInstrument(name)

	// Set the global counter atomically
	atomic.StorePointer(&globalLogCounter, unsafe.Pointer(counter))

	for _, level := range []logging.Level{logging.Debug, logging.Info, logging.Warn, logging.Error} {
		m.AddInt(ctx, name, 0, telemetry.InstAttribs{
			{Name: telemetry.AttribLogLevel, Value: logLevelName(level)},
		})
	}

	return err
}
