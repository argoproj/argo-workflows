package logging

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2"
)

// go-logr levels don't match our logging levels, so we need to map them
const (
	logrInfoLevel  = 0 // Info level in logr
	logrDebugLevel = 1 // Debug level starts at 1 in logr
	logrMaxLevel   = 4 // Maximum supported debug level
)

// SetupKlogAdapter configures klog to route through our logging system,
// ensuring klog output (from client-go informers, etc.) respects the
// configured log format (e.g. JSON).
func SetupKlogAdapter(ctx context.Context) {
	logger := RequireLoggerFromContext(ctx)
	sink := &logrSink{logger: logger}
	klog.SetLogger(logr.New(sink))
}

// logrSink adapts our logging system to logr's LogSink interface
type logrSink struct {
	logger Logger
}

// Init implements logr.LogSink
func (s *logrSink) Init(info logr.RuntimeInfo) {
	// No initialization needed
}

// Enabled implements logr.LogSink
func (s *logrSink) Enabled(level int) bool {
	return s.isLevelEnabled(level)
}

// Info implements logr.LogSink
func (s *logrSink) Info(level int, msg string, keysAndValues ...any) {
	fields := s.parseKeyValues(keysAndValues)
	loggerWithFields := s.logger.WithFields(fields)
	s.logAtLevel(loggerWithFields, level, msg)
}

// Error implements logr.LogSink
func (s *logrSink) Error(err error, msg string, keysAndValues ...any) {
	fields := s.parseKeyValues(keysAndValues)
	loggerWithFields := s.logger.WithFields(fields)
	if err != nil {
		loggerWithFields = loggerWithFields.WithError(err)
	}
	loggerWithFields.Error(context.Background(), msg)
}

// WithName implements logr.LogSink
func (s *logrSink) WithName(name string) logr.LogSink {
	return &logrSink{
		logger: s.logger.WithField("logger", name),
	}
}

// WithValues implements logr.LogSink
func (s *logrSink) WithValues(keysAndValues ...any) logr.LogSink {
	fields := s.parseKeyValues(keysAndValues)
	return &logrSink{
		logger: s.logger.WithFields(fields),
	}
}

// parseKeyValues converts logr key-value pairs to our Fields format
func (s *logrSink) parseKeyValues(keysAndValues []any) Fields {
	fields := make(Fields)
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			if key, ok := keysAndValues[i].(string); ok {
				fields[key] = keysAndValues[i+1]
			}
		}
	}
	return fields
}

// isLevelEnabled checks if a logr level should be logged based on our logger's level
func (s *logrSink) isLevelEnabled(level int) bool {
	switch level {
	case logrInfoLevel:
		return true // Info level - always enabled
	case logrDebugLevel, 2, 3, logrMaxLevel:
		return s.logger.Level() == Debug // Debug level for higher verbosity
	default:
		return false
	}
}

// logAtLevel maps logr levels to our logging levels and logs the message
func (s *logrSink) logAtLevel(logger Logger, level int, msg string) {
	switch level {
	case logrDebugLevel, 2, 3, logrMaxLevel:
		logger.Debug(context.Background(), msg)
	default:
		logger.Info(context.Background(), msg)
	}
}
