package logging

import "context"

const (
	ErrorField string = "error"
)

type Fields map[string]interface{}

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
}
