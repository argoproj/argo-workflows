package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
)

type slogLogger struct {
	fields    Fields
	logger    *slog.Logger
	level     Level
	hooks     map[Level][]Hook
	withPanic bool
	withFatal bool
}

func fieldsToAttrs(fields Fields) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(fields))
	for k, v := range fields {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

// NewSlogLogger returns a slog based logger
func NewSlogLogger(logLevel Level, format LogType, hooks ...Hook) Logger {
	return NewSlogLoggerCustom(logLevel, format, os.Stderr, hooks...)
}

// NewSlogLoggerCustom returns a slog based logger with custom output destination
func NewSlogLoggerCustom(logLevel Level, format LogType, out io.Writer, hooks ...Hook) Logger {
	var handler slog.Handler
	if logLevel == "" {
		panic("logLevel is required")
	}
	if format == "" {
		panic("format is required")
	}

	mappedHooks := make(map[Level][]Hook)

	// Include global hooks with any additional hooks
	allHooks := append(GetGlobalHooks(), hooks...)
	for _, hook := range allHooks {
		levels := hook.Levels()
		for _, level := range levels {
			mappedHooks[level] = append(mappedHooks[level], hook)
		}
	}

	switch format {
	case Text:
		handler = slog.NewTextHandler(out, &slog.HandlerOptions{Level: convertLevel(logLevel)})
	default:
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: convertLevel(logLevel)})
	}

	f := make(Fields)
	l := slog.New(handler)
	s := slogLogger{
		fields: f,
		logger: l,
		level:  logLevel,
		hooks:  mappedHooks,
	}

	emitInitLogs(context.Background(), &s)
	return &s
}

func (s *slogLogger) Level() Level {
	return s.level
}

func (s *slogLogger) WithFields(fields Fields) Logger {
	newFields := make(Fields)

	maps.Copy(newFields, s.fields)

	maps.Copy(newFields, fields)

	return &slogLogger{
		fields:    newFields,
		logger:    s.logger,
		level:     s.level,
		hooks:     s.hooks,
		withFatal: s.withFatal,
		withPanic: s.withPanic,
	}
}

func (s *slogLogger) WithField(name string, value any) Logger {
	newFields := make(Fields)

	maps.Copy(newFields, s.fields)

	newFields[name] = value

	return &slogLogger{
		fields:    newFields,
		logger:    s.logger,
		level:     s.level,
		hooks:     s.hooks,
		withFatal: s.withFatal,
		withPanic: s.withPanic,
	}
}

// Only works with Error()
func (s *slogLogger) WithPanic() Logger {
	return &slogLogger{
		fields:    s.fields,
		logger:    s.logger,
		level:     s.level,
		hooks:     s.hooks,
		withPanic: true,
		withFatal: s.withFatal,
	}
}

// Only works with Error()
func (s *slogLogger) WithFatal() Logger {
	return &slogLogger{
		fields:    s.fields,
		logger:    s.logger,
		level:     s.level,
		hooks:     s.hooks,
		withFatal: true,
		withPanic: s.withPanic,
	}
}

func (s *slogLogger) WithError(err error) Logger {
	return s.WithField(ErrorField, err)
}

// executeHooks safely executes hooks with panic recovery
func (s *slogLogger) executeHooks(ctx context.Context, level Level, msg string) {
	switch s.hooks[level] {
	case nil:
		return
	default:
		for _, hook := range s.hooks[level] {
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Log the panic but don't crash the logger
						s.logger.ErrorContext(ctx, "hook panic recovered", "panic", r, "hook", fmt.Sprintf("%T", hook))
					}
				}()
				hook.Fire(ctx, level, msg, s.fields)
			}()
		}
	}
}

func (s *slogLogger) commonLog(ctx context.Context, level Level, msg string) {
	s.executeHooks(ctx, level, msg)
	s.logger.LogAttrs(ctx, convertLevel(level), msg, fieldsToAttrs(s.fields)...)
	switch {
	case s.withFatal:
		exitFunc := GetExitFunc()
		if exitFunc == nil {
			os.Exit(1)
		}
		exitFunc(1)
	case s.withPanic:
		panic(msg)
	}
}

func (s *slogLogger) Debug(ctx context.Context, msg string) {
	s.commonLog(ctx, Debug, msg)
}

func (s *slogLogger) Info(ctx context.Context, msg string) {
	s.commonLog(ctx, Info, msg)
}

func (s *slogLogger) Warn(ctx context.Context, msg string) {
	s.commonLog(ctx, Warn, msg)
}

func (s *slogLogger) Error(ctx context.Context, msg string) {
	s.commonLog(ctx, Error, msg)
}

// convertLevel converts our Level type to slog.Level
func convertLevel(level Level) slog.Level {
	switch level {
	case Debug:
		return slog.LevelDebug
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (s *slogLogger) InContext(ctx context.Context) (context.Context, Logger) {
	return WithLogger(ctx, s), s
}

// NewBackgroundContext returns a new context with this logger in it
func (s *slogLogger) NewBackgroundContext() context.Context {
	return WithLogger(context.Background(), s)
}
