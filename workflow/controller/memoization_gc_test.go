package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

type testMemoizationDB struct {
	pruned   chan struct{}
	pruneErr error
	disabled bool
	checked  chan struct{}
}

func (*testMemoizationDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	return nil, nil
}

func (*testMemoizationDB) Save(context.Context, string, string, string, string, *wfv1.Outputs, string) error {
	return nil
}

func (t *testMemoizationDB) Prune(context.Context) (int64, error) {
	select {
	case t.pruned <- struct{}{}:
	default:
	}
	return 1, t.pruneErr
}

func (t *testMemoizationDB) IsEnabled() bool {
	if t.checked != nil {
		select {
		case t.checked <- struct{}{}:
		default:
		}
	}
	return !t.disabled
}

func TestMemoizationCacheGarbageCollectorDisabled(t *testing.T) {
	for _, period := range []string{"0", "-1"} {
		t.Run(period, func(t *testing.T) {
			t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", period)
			ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
			defer cancel()
			controller := &WorkflowController{}
			done := make(chan struct{})
			go func() {
				defer close(done)
				controller.memoizationCacheGarbageCollector(ctx)
			}()
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("disabled GC should exit without waiting for cancellation")
			}
		})
	}
}

func TestMemoizationCacheGarbageCollectorRejectsInvalidPeriod(t *testing.T) {
	for _, period := range []string{"24h", "1.5", "invalid", "9223372037"} {
		t.Run(period, func(t *testing.T) {
			t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", period)
			controller := &WorkflowController{}
			require.Panics(t, func() {
				controller.memoizationCacheGarbageCollector(logging.TestContext(t.Context()))
			})
		})
	}
}

func TestMemoizationCacheGarbageCollectorContinuesAfterError(t *testing.T) {
	t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", "1")
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	queries := &testMemoizationDB{pruned: make(chan struct{}, 2), pruneErr: errors.New("prune failed")}
	controller := &WorkflowController{}
	controller.setMemoizationQueries(queries)
	done := make(chan struct{})
	go func() {
		defer close(done)
		controller.memoizationCacheGarbageCollector(ctx)
	}()
	for range 2 {
		select {
		case <-queries.pruned:
		case <-time.After(3 * time.Second):
			t.Fatal("GC must continue after a pruning failure")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GC must stop on cancellation after a pruning failure")
	}
}

func TestMemoizationCacheGarbageCollectorHandlesRuntimeEnable(t *testing.T) {
	t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", "1")

	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()

	controller := &WorkflowController{}
	checked := make(chan struct{}, 1)
	controller.setMemoizationQueries(&testMemoizationDB{disabled: true, checked: checked})
	done := make(chan struct{})
	start := time.Now()
	go func() {
		defer close(done)
		controller.memoizationCacheGarbageCollector(ctx)
	}()

	select {
	case <-checked:
		assert.GreaterOrEqual(t, time.Since(start), time.Second, "the period is measured in seconds")
	case <-time.After(3 * time.Second):
		t.Fatal("expected GC to check the initially disabled backend")
	}

	queries := &testMemoizationDB{pruned: make(chan struct{}, 1)}
	controller.setMemoizationQueries(queries)

	select {
	case <-queries.pruned:
	case <-time.After(3 * time.Second):
		t.Fatal("expected memoization cache GC to observe runtime enablement and prune")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected memoization cache GC goroutine to stop after context cancellation")
	}
}
