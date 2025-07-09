package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

type slogLogger struct {
	fields Fields
	logger *slog.Logger
	hooks  map[Level][]Hook
	mu     sync.RWMutex
}

var (
	lock = &sync.Mutex{}
)

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

	mappedHoooks := make(map[Level][]Hook)

	for _, hook := range hooks {
		levels := hook.Levels()
		for _, level := range levels {
			mappedHoooks[level] = append(mappedHoooks[level], hook)
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
		hooks:  mappedHoooks,
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

	// Copy hooks map
	s.mu.RLock()
	newHooks := make(map[Level][]Hook)
	for level, hooks := range s.hooks {
		newHooks[level] = make([]Hook, len(hooks))
		copy(newHooks[level], hooks)
	}
	s.mu.RUnlock()

	return &slogLogger{
		fields: newFields,
		logger: s.logger,
		hooks:  newHooks,
		mu:     sync.RWMutex{},
	}
}

func (s *slogLogger) WithField(name string, value any) Logger {
	newFields := make(Fields)

	for k, v := range s.fields {
		newFields[k] = v
	}

	newFields[name] = value

	// Copy hooks map
	s.mu.RLock()
	newHooks := make(map[Level][]Hook)
	for level, hooks := range s.hooks {
		newHooks[level] = make([]Hook, len(hooks))
		copy(newHooks[level], hooks)
	}
	s.mu.RUnlock()

	return &slogLogger{
		fields: newFields,
		logger: s.logger,
		hooks:  newHooks,
		mu:     sync.RWMutex{},
	}
}

func (s *slogLogger) WithError(err error) Logger {
	return s.WithField(ErrorField, err)
}

// executeHooks safely executes hooks with panic recovery
func (s *slogLogger) executeHooks(ctx context.Context, hooks []Hook, level Level, msg string) {
	for _, hook := range hooks {
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

func (s *slogLogger) Info(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Info]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Info, msg)
	s.logger.LogAttrs(ctx, slog.LevelInfo, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Infof(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Info(ctx, msg)
}

func (s *slogLogger) Warn(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Warn]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Warn, msg)
	s.logger.LogAttrs(ctx, slog.LevelWarn, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Warnf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Warn(ctx, msg)
}

func (s *slogLogger) Fatal(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Fatal]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Fatal, msg)
	s.logger.LogAttrs(ctx, slog.LevelError, msg, fieldsToAttrs(s.fields)...)
	os.Exit(1)
}

func (s *slogLogger) Fatalf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Fatal(ctx, msg)
}

func (s *slogLogger) Error(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Error]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Error, msg)
	s.logger.LogAttrs(ctx, slog.LevelError, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Errorf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Error(ctx, msg)
}

func (s *slogLogger) Debug(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Debug]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Debug, msg)
	s.logger.LogAttrs(ctx, slog.LevelDebug, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Debugf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Debug(ctx, msg)
}

func (s *slogLogger) Warning(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Warn]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Warn, msg)
	s.logger.LogAttrs(ctx, slog.LevelWarn, msg, fieldsToAttrs(s.fields)...)
}

func (s *slogLogger) Warningf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Warning(ctx, msg)
}

func (s *slogLogger) Panic(ctx context.Context, msg string) {
	s.mu.RLock()
	hooks := s.hooks[Panic]
	s.mu.RUnlock()
	if hooks == nil {
		hooks = []Hook{}
	}
	s.executeHooks(ctx, hooks, Panic, msg)
	s.logger.LogAttrs(ctx, slog.LevelError, msg, fieldsToAttrs(s.fields)...)
	panic(msg)
}

func (s *slogLogger) Panicf(ctx context.Context, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	s.Panic(ctx, msg)
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
	case Fatal:
		return slog.LevelError
	case Panic:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
