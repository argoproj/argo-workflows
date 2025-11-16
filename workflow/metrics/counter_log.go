package metrics

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/telemetry"
)

type logMetricsHelper struct {
	addLogMessages func(ctx context.Context, val int64, logLevel string)
}

var globalLogMetrics unsafe.Pointer // *logMetricsHelper

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
	// Get the metrics helper atomically
	helperPtr := (*logMetricsHelper)(atomic.LoadPointer(&globalLogMetrics))

	// Only fire if we have real metrics
	if helperPtr != nil && helperPtr.addLogMessages != nil {
		helperPtr.addLogMessages(ctx, 1, logLevelName(level))
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

	// Set the global metrics helper atomically
	helper := &logMetricsHelper{
		addLogMessages: m.AddLogMessages,
	}
	atomic.StorePointer(&globalLogMetrics, unsafe.Pointer(helper))

	for _, level := range []logging.Level{logging.Debug, logging.Info, logging.Warn, logging.Error} {
		m.AddLogMessages(ctx, 0, logLevelName(level))
	}

	return err
}
