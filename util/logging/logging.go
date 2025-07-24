package logging

import (
	"context"
	"fmt"
	"os"
	"runtime"
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

func TypeFromStringOr(s string, defaultType LogType) (LogType, error) {
	if s == "" {
		return defaultType, nil
	}
	return TypeFromString(s)
}

func TypeFromString(s string) (LogType, error) {
	switch strings.ToLower(s) {
	case "json":
		return JSON, nil
	case "text":
		return Text, nil
	default:
		return Text, fmt.Errorf("invalid log type: %s", s)
	}
}

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

func ParseLevelOr(s string, defaultLevel Level) (Level, error) {
	if s == "" {
		return defaultLevel, nil
	}
	return ParseLevel(s)
}

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

var (
	lock = sync.RWMutex{}

	exitFunc    func(int)
	globalHooks []Hook
)

// SetExitFunc sets the exit function for testing purposes
func SetExitFunc(f func(int)) {
	lock.Lock()
	defer lock.Unlock()
	exitFunc = f
}

// GetExitFunc returns the current exit function
func GetExitFunc() func(int) {
	lock.RLock()
	defer lock.RUnlock()
	return exitFunc
}

// AddGlobalHook adds a hook that will be included in all new loggers
func AddGlobalHook(hook Hook) {
	lock.Lock()
	defer lock.Unlock()
	globalHooks = append(globalHooks, hook)
	// // Recreate the default logger to include the new hook
	// defaultLogger = newSlogLogger(globalLevel, globalFormat)
}

// GetGlobalHooks returns all global hooks
func GetGlobalHooks() []Hook {
	return globalHooks
}

// Fields are used to carry the values of each field
type Fields map[string]any

// Hook is used to tap into the log
type Hook interface {
	Levels() []Level
	Fire(ctx context.Context, level Level, msg string, fields Fields)
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

	// NewBackgroundContext returns a new context with this logger in it
	NewBackgroundContext() context.Context

	// InContext returns a new context with this logger in it
	InContext(ctx context.Context) (context.Context, Logger)

	Level() Level
}

// RequireLoggerFromContext returns a logger from context, panics if not found
// This should be used almost
func RequireLoggerFromContext(ctx context.Context) Logger {
	val := getLoggerFromContext(ctx)
	if val == nil {
		const size = 64 << 10
		stackTraceBuffer := make([]byte, size)
		stackSize := runtime.Stack(stackTraceBuffer, false)
		// Free up the unused spaces
		stackTraceBuffer = stackTraceBuffer[:stackSize]
		fmt.Fprintf(os.Stderr, "no logger in context Call stack:\n%s",
			stackTraceBuffer)

		panic("logger not found in context")
	}
	return val
}

// GetLoggerFromContextOrNil returns a logger from context, returns nil if not found
// You probably should use one of the other functions that return a logger instead of this one
func GetLoggerFromContextOrNil(ctx context.Context) Logger {
	return getLoggerFromContext(ctx)
}

// GetLoggerFromContext returns a logger from context, returns nil if not found
func getLoggerFromContext(ctx context.Context) Logger {
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
