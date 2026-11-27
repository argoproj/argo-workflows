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
	controllercache "github.com/argoproj/argo-workflows/v4/workflow/controller/cache"
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
func (q *testMemoizationDB) Prune(context.Context) (int64, error) {
	select {
	case q.pruned <- struct{}{}:
	default:
	}
	return 1, q.pruneErr
}
func (q *testMemoizationDB) IsEnabled() bool {
	if q.checked != nil {
		select {
		case q.checked <- struct{}{}:
		default:
		}
	}
	return !q.disabled
}

func TestMemoizationGCPeriodFromEnv(t *testing.T) {
	for value, want := range map[string]time.Duration{"": 24 * time.Hour, "0": 0, "-1": 0, "1": time.Second, "86400": 24 * time.Hour} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", value)
			period, err := memoizationGCPeriodFromEnv()
			require.NoError(t, err)
			assert.Equal(t, want, period)
		})
	}
}

func TestNewControllerRejectsInvalidMemoizationGCPeriod(t *testing.T) {
	for _, value := range []string{"24h", "1.5", "invalid", "9223372037"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", value)
			controller, err := NewWorkflowController(logging.TestContext(t.Context()), nil, nil, nil, "", "", "", "", "", "", false, false)
			require.ErrorContains(t, err, "MEMO_CACHE_GC_PERIOD_SECONDS")
			assert.Nil(t, controller, "invalid GC configuration fails before constructing clients or opening databases")
		})
	}
}

func TestMemoizationCacheGarbageCollectorDisabled(t *testing.T) {
	for _, period := range []time.Duration{0, -time.Second} {
		t.Run(period.String(), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(logging.TestContext(t.Context()), time.Second)
			defer cancel()
			q := &testMemoizationDB{pruned: make(chan struct{}, 1)}
			c := &WorkflowController{memoCacheGCPeriod: period, cacheFactory: controllercache.NewCacheFactory(nil, "default")}
			c.cacheFactory.SetQueries(ctx, q)
			c.memoizationCacheGarbageCollector(ctx)
			assert.Empty(t, q.pruned)
		})
	}
}

func runMemoGC(ctx context.Context, t *testing.T, c *WorkflowController) chan struct{} {
	t.Helper()
	done := make(chan struct{})
	go func() { defer close(done); c.memoizationCacheGarbageCollector(ctx) }()
	return done
}

func TestMemoizationCacheGarbageCollectorPrunesImmediately(t *testing.T) {
	// Environment settings have already been parsed by the constructor.
	t.Setenv("MEMO_CACHE_GC_PERIOD_SECONDS", "invalid")
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	q := &testMemoizationDB{pruned: make(chan struct{}, 1)}
	c := &WorkflowController{memoCacheGCPeriod: 24 * time.Hour, cacheFactory: controllercache.NewCacheFactory(nil, "default")}
	c.cacheFactory.SetQueries(ctx, q)
	done := runMemoGC(ctx, t, c)
	select {
	case <-q.pruned:
	case <-time.After(time.Second):
		t.Fatal("first prune must not wait 24 hours")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GC did not stop")
	}
}

func TestMemoizationCacheGarbageCollectorContinuesAfterError(t *testing.T) {
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	q := &testMemoizationDB{pruned: make(chan struct{}, 2), pruneErr: errors.New("prune failed")}
	c := &WorkflowController{memoCacheGCPeriod: time.Millisecond, cacheFactory: controllercache.NewCacheFactory(nil, "default")}
	c.cacheFactory.SetQueries(ctx, q)
	done := runMemoGC(ctx, t, c)
	for range 2 {
		select {
		case <-q.pruned:
		case <-time.After(time.Second):
			t.Fatal("GC must continue after a pruning failure")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GC did not stop")
	}
}

func TestMemoizationCacheGarbageCollectorHandlesRuntimeEnable(t *testing.T) {
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()
	checked := make(chan struct{}, 1)
	c := &WorkflowController{memoCacheGCPeriod: time.Millisecond, cacheFactory: controllercache.NewCacheFactory(nil, "default")}
	c.cacheFactory.SetQueries(ctx, &testMemoizationDB{disabled: true, checked: checked})
	done := runMemoGC(ctx, t, c)
	select {
	case <-checked:
	case <-time.After(time.Second):
		t.Fatal("GC did not check disabled backend")
	}
	q := &testMemoizationDB{pruned: make(chan struct{}, 1)}
	c.cacheFactory.SetQueries(ctx, q)
	select {
	case <-q.pruned:
	case <-time.After(time.Second):
		t.Fatal("GC did not observe enabled backend")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("GC did not stop")
	}
}
