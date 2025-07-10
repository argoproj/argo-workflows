package logging

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	// ErrorField is the default name for a WithError call
	ErrorField string = "error"
)

// CtxKey contains context keys for this package
type CtxKey string

const (
	// LoggerKey is used to obtain/set the logger from a context
	LoggerKey CtxKey = "logger"
)

type LogType string

const (
	JSON LogType = "json"
	Text LogType = "text"
)

var (
	lock = sync.RWMutex{}

	globalLevel   = Info
	globalFormat  = Text
	defaultLogger = NewSlogLogger(globalLevel, globalFormat)
)

// SetGlobalLevel sets the global log level
func SetGlobalLevel(level Level) {
	lock.Lock()
	defer lock.Unlock()
	globalLevel = level
	defaultLogger = NewSlogLogger(globalLevel, globalFormat)
}

// GetGlobalLevel returns the current global log level
func GetGlobalLevel() Level {
	lock.RLock()
	defer lock.RUnlock()
	return globalLevel
}

// SetGlobalFormat sets the global log format
func SetGlobalFormat(format LogType) {
	lock.Lock()
	defer lock.Unlock()
	globalFormat = format
	defaultLogger = NewSlogLogger(globalLevel, globalFormat)
}

// GetGlobalFormat returns the current global log format
func GetGlobalFormat() LogType {
	lock.RLock()
	defer lock.RUnlock()
	return globalFormat
}

// GetDefaultLogger returns the default logger configured with global settings
func GetDefaultLogger() Logger {
	lock.RLock()
	defer lock.RUnlock()
	return defaultLogger
}

// Fields are used to carry the values of each field
type Fields map[string]any

// Level is used to indicate log level
type Level string

const (
	// Debug level events
	Debug Level = "debug"
	// Info level events
	Info Level = "info"
	// Warn level events
	Warn Level = "warn"
	// Error level events
	Error Level = "error"
)

// ParseLevel parses a string into a Level enum
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	// legacy removed level
	case "trace":
		return Debug, nil
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	// legacy removed level
	case "print":
		return Info, nil
	case "warn":
		return Warn, nil
	case "error":
		return Error, nil
	// legacy removed level
	case "fatal":
		return Error, nil
	// legacy removed level
	case "panic":
		return Error, nil
	default:
		return "", fmt.Errorf("invalid log level: %s", s)
	}
}

// Hook is used to tap into the log
type Hook interface {
	Levels() []Level
	Fire(level Level, msg string)
}

// Logger exports a logging interface
type Logger interface {
	WithFields(fields Fields) Logger
	WithField(name string, value any) Logger
	WithError(err error) Logger

	// When issuing Error, adding this will Panic
	WithPanic() Logger
	// When issuing Error, adding this will exit 1
	WithFatal() Logger

	Debug(ctx context.Context, msg string)
	Debugf(ctx context.Context, format string, args ...any)

	Info(ctx context.Context, msg string)
	Infof(ctx context.Context, format string, args ...any)

	Warn(ctx context.Context, msg string)
	Warnf(ctx context.Context, format string, args ...any)

	Error(ctx context.Context, msg string)
	Errorf(ctx context.Context, format string, args ...any)
}

// GetLoggerFromContext returns a logger from context, returns nil if not found
func GetLoggerFromContext(ctx context.Context) Logger {
	val := ctx.Value(LoggerKey)
	if val == nil {
		return nil
	}
	return val.(Logger)
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}
