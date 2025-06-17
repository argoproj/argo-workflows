package logging

import "context"

const (
	// ErrorField is the default name for a WithError call
	ErrorField string = "error"
)

// Fields are used to carry the values of each field
type Fields map[string]interface{}

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

// Hook is used to tap into the log
type Hook interface {
	Levels() []Level
	Fire(msg string)
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
