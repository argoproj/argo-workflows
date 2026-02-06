package logging

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/klog/v2"
)

func TestSetupKlogAdapter(t *testing.T) {
	t.Run("klog output respects JSON format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewSlogLoggerCustom(Info, JSON, &buf)
		ctx := WithLogger(context.Background(), logger)

		SetupKlogAdapter(ctx)

		klog.Info("test message from klog")

		output := buf.String()
		assert.Contains(t, output, "{", "klog output should be JSON formatted, got: %s", output)
		assert.Contains(t, output, "test message from klog", "klog output should contain the message, got: %s", output)
	})

	t.Run("klog output respects text format", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewSlogLoggerCustom(Info, Text, &buf)
		ctx := WithLogger(context.Background(), logger)

		SetupKlogAdapter(ctx)

		klog.Info("text format message")

		output := buf.String()
		assert.Contains(t, output, "text format message", "klog output should contain the message, got: %s", output)
		assert.False(t, strings.HasPrefix(output, "{"), "klog output should not be JSON formatted, got: %s", output)
	})

	t.Run("klog error output includes error", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewSlogLoggerCustom(Info, JSON, &buf)
		ctx := WithLogger(context.Background(), logger)

		SetupKlogAdapter(ctx)

		klog.Error("something went wrong")

		output := buf.String()
		assert.Contains(t, output, "something went wrong", "klog error output should contain the message, got: %s", output)
	})

	t.Run("klog V(1) output filtered at default verbosity", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewSlogLoggerCustom(Debug, JSON, &buf)
		ctx := WithLogger(context.Background(), logger)

		SetupKlogAdapter(ctx)

		// klog.V(1) is filtered by klog's own verbosity flag (default 0)
		// before reaching our adapter - this is expected behavior
		klog.V(1).Info("debug message should be filtered by klog verbosity")

		output := buf.String()
		assert.Empty(t, output, "V(1) messages should be filtered when klog verbosity is 0")
	})
}
