package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetInitStorage resets the global init log storage for test isolation
func resetInitStorage() {
	initStorage.mutex.Lock()
	defer initStorage.mutex.Unlock()
	initStorage.initLogs = nil
	initStorage.fatal = false
}

func TestInitLogger(t *testing.T) {
	t.Run("basic logging", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := context.Background()

		// Log some messages during initialization
		InitLogger().Info(ctx, "starting application")
		InitLogger().WithField("version", "1.0.0").Info(ctx, "version info")
		InitLogger().WithError(assert.AnError).Warn(ctx, "warning during init")
		InitLogger().Debug(ctx, "debug message")

		// Create the real logger, which emits init logs
		_ = NewSlogLoggerCustom(Debug, JSON, &buf)

		// Verify output contains all logged messages
		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 4)

		var logEntry map[string]any

		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "starting application", logEntry["msg"])
		assert.Equal(t, "INFO", logEntry["level"])

		err = json.Unmarshal([]byte(lines[1]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "version info", logEntry["msg"])
		assert.Equal(t, "INFO", logEntry["level"])
		assert.Equal(t, "1.0.0", logEntry["version"])

		err = json.Unmarshal([]byte(lines[2]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "warning during init", logEntry["msg"])
		assert.Equal(t, "WARN", logEntry["level"])
		assert.NotNil(t, logEntry["error"])

		err = json.Unmarshal([]byte(lines[3]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "debug message", logEntry["msg"])
		assert.Equal(t, "DEBUG", logEntry["level"])
	})

	t.Run("debug level shows debug messages", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		initLogger.Debug(ctx, "debug message")
		initLogger.Info(ctx, "info message")

		_ = NewSlogLoggerCustom(Debug, JSON, &buf)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 2)

		var logEntry map[string]any

		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "debug message", logEntry["msg"])
		assert.Equal(t, "DEBUG", logEntry["level"])

		err = json.Unmarshal([]byte(lines[1]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "info message", logEntry["msg"])
		assert.Equal(t, "INFO", logEntry["level"])
	})

	t.Run("text format", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		initLogger.Debug(ctx, "debug message") // should not be logged
		initLogger.WithField("key", "value").Warn(ctx, "warning with field")
		initLogger.Info(ctx, "text format message")

		_ = NewSlogLoggerCustom(Info, Text, &buf)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 2)

		assert.Contains(t, lines[0], "warning with field")
		assert.Contains(t, lines[0], "WARN")
		assert.Contains(t, lines[0], "key=value")
		assert.Contains(t, lines[1], "text format message")
		assert.Contains(t, lines[1], "INFO")
		assert.NotContains(t, lines[1], "key=value")
	})

	t.Run("fields are preserved", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		initLogger.WithField("service", "test").
			WithField("instance", "123").
			WithError(assert.AnError).
			Info(ctx, "message with multiple fields")

		_ = NewSlogLoggerCustom(Info, JSON, &buf)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 1)

		var logEntry map[string]any
		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)

		assert.Equal(t, "message with multiple fields", logEntry["msg"])
		assert.Equal(t, "test", logEntry["service"])
		assert.Equal(t, "123", logEntry["instance"])
		assert.NotNil(t, logEntry["error"])
	})

	t.Run("multiple init loggers share storage", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := InitLoggerInContext()
		initLogger1 := RequireLoggerFromContext(ctx)
		initLogger2 := InitLogger()

		initLogger1.Info(ctx, "message from logger 1")
		initLogger2.WithField("source", "logger2").Info(ctx, "message from logger 2")

		_ = NewSlogLoggerCustom(Info, JSON, &buf)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 2)

		var logEntry map[string]any

		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "message from logger 1", logEntry["msg"])

		err = json.Unmarshal([]byte(lines[1]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "message from logger 2", logEntry["msg"])
		assert.Equal(t, "logger2", logEntry["source"])
	})

	t.Run("panic on unsupported methods", func(t *testing.T) {
		// These are very deliberately unsupported methods, so we should panic if they are called.
		resetInitStorage()
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		assert.Panics(t, func() {
			initLogger.WithPanic()
		})

		assert.Panics(t, func() {
			//nolint:contextchec
			initLogger.NewBackgroundContext()
		})

		assert.Panics(t, func() {
			initLogger.InContext(ctx)
		})

		assert.Panics(t, func() {
			initLogger.Level()
		})
	})

	t.Run("WithFatal causes emit init logs and exit on error", func(t *testing.T) {
		resetInitStorage()
		var bufInit bytes.Buffer
		var bufNormal bytes.Buffer
		ctx := context.Background()

		// Set up a custom exit function to capture the exit call
		exitCalled := false
		exitCode := 0
		originalExitFunc := GetExitFunc()
		defer SetExitFunc(originalExitFunc)

		SetExitFunc(func(code int) {
			exitCalled = true
			exitCode = code
		})

		initStorage.out = &bufInit
		// Log some messages first
		InitLogger().Info(ctx, "info message before fatal")
		InitLogger().WithField("key", "value").Warn(ctx, "warning before fatal")

		// Create a logger with WithFatal and call Error
		InitLogger().WithFatal().WithField("fatal_field", "fatal_value").Error(ctx, "fatal error message")

		// Verify that exit was called with code 1
		assert.True(t, exitCalled, "Exit function should have been called")
		assert.Equal(t, 1, exitCode, "Exit should be called with code 1")

		output := bufInit.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 3)

		var logEntry map[string]any

		err := json.Unmarshal([]byte(lines[0]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "info message before fatal", logEntry["msg"])
		assert.Equal(t, "INFO", logEntry["level"])

		err = json.Unmarshal([]byte(lines[1]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "warning before fatal", logEntry["msg"])
		assert.Equal(t, "WARN", logEntry["level"])
		assert.Equal(t, "value", logEntry["key"])

		err = json.Unmarshal([]byte(lines[2]), &logEntry)
		require.NoError(t, err)
		assert.Equal(t, "fatal error message", logEntry["msg"])
		assert.Equal(t, "ERROR", logEntry["level"])
		assert.Equal(t, "fatal_value", logEntry["fatal_field"])

		// Verify that init logs were emitted before exit, we should not see any logs
		_ = NewSlogLoggerCustom(Info, JSON, &bufNormal)
		output = bufNormal.String()
		lines = strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 1)
	})

	t.Run("concurrent access", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		// Start multiple goroutines logging simultaneously
		done := make(chan bool, 10)
		for i := range 10 {
			go func(id int) {
				defer func() { done <- true }()
				initLogger.WithField("goroutine", id).Info(ctx, "concurrent message")
			}(i)
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}

		_ = NewSlogLoggerCustom(Info, JSON, &buf)

		output := buf.String()
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 10)

		for _, line := range lines {
			var logEntry map[string]any
			err := json.Unmarshal([]byte(line), &logEntry)
			require.NoError(t, err)
			assert.Equal(t, "concurrent message", logEntry["msg"])
			assert.NotNil(t, logEntry["goroutine"])
		}
	})

	t.Run("hook execution during init emission", func(t *testing.T) {
		resetInitStorage()
		var buf bytes.Buffer
		var hookCalls []string
		ctx := InitLoggerInContext()
		initLogger := RequireLoggerFromContext(ctx)

		initLogger.Info(ctx, "message with hook")

		testHook := &testHook{
			calls: &hookCalls,
		}
		_ = NewSlogLoggerCustom(Info, JSON, &buf, testHook)

		require.Len(t, hookCalls, 1)
		assert.Equal(t, "message with hook", hookCalls[0])
	})

	t.Run("hook receives fields", func(t *testing.T) {
		var buf bytes.Buffer
		var hookFields []Fields
		ctx := context.Background()

		testHook := &fieldsTestHook{
			fields: &hookFields,
		}
		logger := NewSlogLoggerCustom(Info, JSON, &buf, testHook)

		logger.WithField("service", "test").
			WithField("instance", "123").
			WithError(assert.AnError).
			Info(ctx, "message with fields")

		require.Len(t, hookFields, 1)
		assert.Equal(t, "test", hookFields[0]["service"])
		assert.Equal(t, "123", hookFields[0]["instance"])
		assert.NotNil(t, hookFields[0]["error"])
	})
}

// testHook is a simple hook for testing
type testHook struct {
	calls *[]string
}

func (h *testHook) Levels() []Level {
	return []Level{Info}
}

func (h *testHook) Fire(ctx context.Context, level Level, message string, fields Fields) {
	*h.calls = append(*h.calls, message)
}

// fieldsTestHook is a hook for testing that captures fields
type fieldsTestHook struct {
	fields *[]Fields
}

func (h *fieldsTestHook) Levels() []Level {
	return []Level{Info}
}

func (h *fieldsTestHook) Fire(ctx context.Context, level Level, message string, fields Fields) {
	*h.fields = append(*h.fields, fields)
}
