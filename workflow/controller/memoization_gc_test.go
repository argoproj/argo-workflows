package controller

import (
	"context"
	"testing"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

type testMemoizationDB struct {
	pruned chan struct{}
}

func (*testMemoizationDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	return nil, nil
}

func (*testMemoizationDB) Save(context.Context, string, string, string, string, *wfv1.Outputs, int64) error {
	return nil
}

func (t *testMemoizationDB) Prune(context.Context) (int64, error) {
	select {
	case t.pruned <- struct{}{}:
	default:
	}
	return 1, nil
}

func (*testMemoizationDB) IsEnabled() bool {
	return true
}

func TestMemoizationCacheGarbageCollectorHandlesRuntimeEnable(t *testing.T) {
	t.Setenv("MEMO_CACHE_GC_PERIOD", "10ms")

	ctx, cancel := context.WithCancel(logging.TestContext(t.Context()))
	defer cancel()

	controller := &WorkflowController{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		controller.memoizationCacheGarbageCollector(ctx)
	}()

	time.Sleep(25 * time.Millisecond)

	queries := &testMemoizationDB{pruned: make(chan struct{}, 1)}
	controller.setMemoizationQueries(queries)

	select {
	case <-queries.pruned:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("expected memoization cache GC to observe runtime enablement and prune")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected memoization cache GC goroutine to stop after context cancellation")
	}
}
