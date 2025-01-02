package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

type slogLogger struct {
	fields Fields
	logger *slog.Logger
}

func (s *slogLogger) WithFields(_ context.Context, fields Fields) Logger {
	logger := s.logger

	newFields := make(Fields)
	for k, v := range s.fields {
		newFields[k] = v
		logger = logger.With(k, v)
	}
	for k, v := range fields {
		newFields[k] = v
		logger = logger.With(k, v)
	}

	return &slogLogger{
		fields: newFields,
		logger: logger,
	}
}

func (s *slogLogger) WithField(_ context.Context, name string, value interface{}) Logger {
	newFields := make(Fields)

	logger := s.logger
	for k, v := range s.fields {
		newFields[k] = v
		logger = s.logger.With(k, v)
	}

	logger = logger.With(name, value)
	newFields[name] = value
	return &slogLogger{
		fields: newFields,
		logger: logger,
	}
}

func (s *slogLogger) WithError(ctx context.Context, err error) Logger {
	return s.WithField(ctx, ErrorField, err)
}

func (s *slogLogger) Info(ctx context.Context, msg string) {
	s.logger.InfoContext(ctx, msg)
}

func (s *slogLogger) Infof(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.InfoContext(ctx, msg)
}

func (s *slogLogger) Warn(ctx context.Context, msg string) {
	s.logger.WarnContext(ctx, msg)
}

func (s *slogLogger) Warnf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.WarnContext(ctx, msg)
}

func (s *slogLogger) Fatal(ctx context.Context, msg string) {
	s.logger.ErrorContext(ctx, msg)
	os.Exit(1)
}

func (s *slogLogger) Fatalf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.ErrorContext(ctx, msg)
	os.Exit(1)
}

func (s *slogLogger) Error(ctx context.Context, msg string) {
	s.logger.ErrorContext(ctx, msg)
}

func (s *slogLogger) Errorf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.ErrorContext(ctx, msg)
}

func (s *slogLogger) Debug(ctx context.Context, msg string) {
	s.logger.DebugContext(ctx, msg)
}

func (s *slogLogger) Debugf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.DebugContext(ctx, msg)
}

func (s *slogLogger) Warning(ctx context.Context, msg string) {
	s.logger.WarnContext(ctx, msg)
}

func (s *slogLogger) Warningf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.WarnContext(ctx, msg)
}

func (s *slogLogger) Println(ctx context.Context, msg string) {
	s.logger.InfoContext(ctx, msg)
}

func (s *slogLogger) Printf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.InfoContext(ctx, msg)
}

func (s *slogLogger) Panic(ctx context.Context, msg string) {
	s.logger.ErrorContext(ctx, msg)
	panic(msg)
}

func (s *slogLogger) Panicf(ctx context.Context, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.logger.ErrorContext(ctx, msg)
	panic(msg)
}

// NewSlogLogger returns a slog based logger
func NewSlogLogger() Logger {
	f := make(Fields)
	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	// l := slog.Default()
	s := slogLogger{
		fields: f,
		logger: l,
	}
	return &s
}
