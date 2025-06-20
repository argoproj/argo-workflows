package logging

import (
	"context"
	"fmt"
	"strings"
)

const (
	// ErrorField is the default name for a WithError call
	ErrorField string = "error"
)

// CtxKey contains context keys for this package
type CtxKey string

const (
	//LoggerKey is used to obtain/set the logger from a context
	LoggerKey CtxKey = "logger"
)

var (
	globalLevel  = Info
	globalFormat = Text
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
	lock.Lock()
	defer lock.Unlock()
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
	lock.Lock()
	defer lock.Unlock()
	return globalFormat
}

// Fields are used to carry the values of each field
type Fields map[string]interface{}

// Level is used to indicate log level
type Level string

const (
	// Trace level events
	Trace Level = "trace"
	// Debug level events
	Debug Level = "debug"
	// Info level events
	Info Level = "info"
	// Warn level events
	Warn Level = "warn"
	// Error level events
	Error Level = "error"
	// Fatal level events
	Fatal Level = "fatal"
	// Print level events
	Print Level = "print"
	// Panic level events
	Panic Level = "panic"
)

// ParseLevel parses a string into a Level enum
func ParseLevel(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "trace":
		return Trace, nil
	case "debug":
		return Debug, nil
	case "info":
		return Info, nil
	case "warn":
		return Warn, nil
	case "error":
		return Error, nil
	case "fatal":
		return Fatal, nil
	case "print":
		return Print, nil
	case "panic":
		return Panic, nil
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
	WithFields(ctx context.Context, fields Fields) Logger
	WithField(ctx context.Context, name string, value interface{}) Logger
	WithError(ctx context.Context, err error) Logger

	Info(ctx context.Context, msg string)
	Infof(ctx context.Context, format string, args ...interface{})

	Warn(ctx context.Context, msg string)
	Warnf(ctx context.Context, format string, args ...interface{})

	Fatal(ctx context.Context, msg string)
	Fatalf(ctx context.Context, format string, args ...interface{})

	Error(ctx context.Context, msg string)
	Errorf(ctx context.Context, format string, args ...interface{})

	Trace(ctx context.Context, msg string)
	Tracef(ctx context.Context, format string, args ...interface{})

	Debug(ctx context.Context, msg string)
	Debugf(ctx context.Context, format string, args ...interface{})

	Warning(ctx context.Context, msg string)
	Warningf(ctx context.Context, format string, args ...interface{})

	Println(ctx context.Context, msg string)
	Printf(ctx context.Context, format string, args ...interface{})

	Panic(ctx context.Context, msg string)
	Panicf(ctx context.Context, format string, args ...interface{})

	AddHook(hook Hook)
}

// GetLoggerFromContext returns a logger from context or the default logger if none is found
func GetLoggerFromContext(ctx context.Context) Logger {
	if ctx == nil {
		return DefaultSlogLogger()
	}

	// Try to get logger from context
	if logger, ok := ctx.Value(LoggerKey).(Logger); ok && logger != nil {
		return logger
	}

	return DefaultSlogLogger()
}

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}
