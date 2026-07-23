package logging

import (
	"context"
	"errors"
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
	format    LogType
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
		format: format,
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
		format:    s.format,
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
		format:    s.format,
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
		format:    s.format,
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
		format:    s.format,
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

// multiHandler fans out each record to multiple slog.Handlers.
type multiHandler struct {
	handlers []slog.Handler
}

func (m *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (m *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range m.handlers {
		// handlers may have different levels, so re-check each rather than relying on multiHandler.Enabled
		if h.Enabled(ctx, r.Level) {
			if err := h.Handle(ctx, r.Clone()); err != nil {
				errs = append(errs, err)
			}
		}
	}
	// attempt all handlers before returning, so one handler's failure does not
	// prevent writing to the others (e.g. stderr write error must not block combined).
	return errors.Join(errs...)
}

func (m *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (m *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// TeeLogger returns a new Logger that writes to both the original destination and w,
// preserving all fields, format and hooks from the original logger.
func TeeLogger(logger Logger, w io.Writer) Logger {
	sl, ok := logger.(*slogLogger)
	if !ok {
		// if an unexpected Logger implementation is passed, return it unchanged instead of panicking
		return logger
	}
	var additionalHandler slog.Handler
	switch sl.format {
	case Text:
		additionalHandler = slog.NewTextHandler(w, &slog.HandlerOptions{Level: convertLevel(sl.level)})
	default:
		additionalHandler = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: convertLevel(sl.level)})
	}
	return &slogLogger{
		fields:    sl.fields,
		logger:    slog.New(&multiHandler{handlers: []slog.Handler{sl.logger.Handler(), additionalHandler}}),
		level:     sl.level,
		format:    sl.format,
		hooks:     sl.hooks,
		withPanic: sl.withPanic,
		withFatal: sl.withFatal,
	}
}
