package cron

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestScheduledTimePreservesConfiguredTimestamp(t *testing.T) {
	want := time.Date(2026, 11, 14, 10, 20, 0, 123, time.UTC)
	woc := &cronWfOperationCtx{
		ctx:               logging.TestContext(t.Context()),
		scheduledTimeFunc: func(_ context.Context) time.Time { return want },
	}

	assert.Equal(t, want, woc.scheduledTime())
}

func TestScheduledTimeFallsBackForZeroTimestamp(t *testing.T) {
	woc := &cronWfOperationCtx{
		ctx:               logging.TestContext(t.Context()),
		scheduledTimeFunc: func(_ context.Context) time.Time { return time.Time{} },
	}

	got := woc.scheduledTime()
	assert.False(t, got.IsZero())
	assert.Equal(t, 0, got.Second())
	assert.Equal(t, 0, got.Nanosecond())
	assert.NotEqual(t, int64(-62135596800), got.Unix())
	assert.NotEqual(t, "workflow--62135596800", getChildWorkflowName("workflow", got))
}
