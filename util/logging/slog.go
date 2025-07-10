package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

type slogLogger struct {
	fields    Fields
	logger    *slog.Logger
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
	return NewSlogLoggerCustom(logLevel, format, os.Stdout, hooks...)
}

// NewSlogLoggerCustom returns a slog based logger with custom output destination
func NewSlogLoggerCustom(logLevel Level, format LogType, out io.Writer, hooks ...Hook) Logger {
	var handler slog.Handler

	mappedHooks := make(map[Level][]Hook)

	for _, hook := range hooks {
		levels := hook.Levels()
		for _, level := range levels {
			mappedHooks[level] = append(mappedHooks[level], hook)
		}
	}

	switch format {
	case JSON:
		handler = slog.NewJSONHandler(out, &slog.HandlerOptions{Level: convertLevel(logLevel)})
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
		hooks:  mappedHooks,
	}
	return &s
}

func (s *slogLogger) WithFields(fields Fields) Logger {
	newFields := make(Fields)

	for k, v := range s.fields {
		newFields[k] = v
	}

	for k, v := range fields {
		newFields[k] = v
	}

	return &slogLogger{
		fields:    newFields,
		logger:    s.logger,
		hooks:     s.hooks,
		withFatal: s.withFatal,
		withPanic: s.withPanic,
	}
}

func (s *slogLogger) WithField(name string, value any) Logger {
	newFields := make(Fields)

	for k, v := range s.fields {
		newFields[k] = v
	}

	newFields[name] = value

	return &slogLogger{
		fields:    newFields,
		logger:    s.logger,
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
				hook.Fire(level, msg)
			}()
		}
	}
}

func (s *slogLogger) Info(ctx context.Context, msg string) {
	s.executeHooks(ctx, Info, msg)
	s.logger.LogAttrs(ctx, slog.LevelInfo, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Infof(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Info(ctx, msg)
}

func (s *slogLogger) Warn(ctx context.Context, msg string) {
	s.executeHooks(ctx, Warn, msg)
	s.logger.LogAttrs(ctx, slog.LevelWarn, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Warnf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Warn(ctx, msg)
}

func (s *slogLogger) Error(ctx context.Context, msg string) {
	s.executeHooks(ctx, Error, msg)
	s.logger.LogAttrs(ctx, slog.LevelError, msg, fieldsToAttrs(s.fields)...)
	switch {
	case s.withFatal:
		os.Exit(1)
	case s.withPanic:
		panic(msg)
	}
}

func (s *slogLogger) Errorf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Error(ctx, msg)
}

func (s *slogLogger) Debug(ctx context.Context, msg string) {
	s.executeHooks(ctx, Debug, msg)
	s.logger.LogAttrs(ctx, slog.LevelDebug, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Debugf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Debug(ctx, msg)
}

// convertLevel converts our Level type to slog.Level
func convertLevel(level Level) slog.Level {
	switch level {
	case Debug:
		return slog.LevelDebug
	case Info:
		return slog.LevelInfo
	case Warn:
		return slog.LevelWarn
	case Error:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
