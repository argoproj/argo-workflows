package cache

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

func TestCacheFactoryNamespacesCachesSeparately(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset())

	cacheA := factory.GetCache(ctx, ConfigMapCache, "ns-a", "shared-cache")
	cacheARepeat := factory.GetCache(ctx, ConfigMapCache, "ns-a", "shared-cache")
	cacheB := factory.GetCache(ctx, ConfigMapCache, "ns-b", "shared-cache")

	require.NotNil(t, cacheA)
	require.NotNil(t, cacheARepeat)
	require.NotNil(t, cacheB)
	assert.Same(t, cacheA, cacheARepeat)
	assert.NotSame(t, cacheA, cacheB)
	assert.Equal(t, "ns-a", cacheA.(*configMapCache).namespace)
	assert.Equal(t, "ns-b", cacheB.(*configMapCache).namespace)
}

func TestCacheFactoryRequiresNamespace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset())

	cache := factory.GetCache(ctx, ConfigMapCache, "", "shared-cache")
	assert.Nil(t, cache)
}

type testMemoizationDB struct {
	enabled   bool
	saveCalls atomic.Int32
	loadStart chan struct{}
	loadBlock chan struct{}
}

func (t *testMemoizationDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	if t.loadStart != nil {
		close(t.loadStart)
	}
	if t.loadBlock != nil {
		<-t.loadBlock
	}
	return nil, nil
}

func (t *testMemoizationDB) Save(context.Context, string, string, string, string, *wfv1.Outputs, int64) error {
	t.saveCalls.Add(1)
	return nil
}

func (*testMemoizationDB) Prune(context.Context) (int64, error) {
	return 0, nil
}

func (t *testMemoizationDB) IsEnabled() bool {
	return t.enabled
}

func TestCacheFactoryStaleSQLCacheNoopsAfterDisable(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset()).(*cacheFactory)
	queries := &testMemoizationDB{enabled: true}
	factory.SetQueries(queries)

	cache := factory.GetCache(ctx, ConfigMapCache, "default", "shared-cache")
	require.NotNil(t, cache)

	factory.SetQueries(nil)

	require.NoError(t, cache.Save(ctx, "memo-key", "node-1", &wfv1.Outputs{}, "1h"))
	assert.Zero(t, queries.saveCalls.Load())
}

func TestCacheFactoryDisableWaitsForInflightSQLLoad(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset()).(*cacheFactory)
	queries := &testMemoizationDB{
		enabled:   true,
		loadStart: make(chan struct{}),
		loadBlock: make(chan struct{}),
	}
	factory.SetQueries(queries)

	cache := factory.GetCache(ctx, ConfigMapCache, "default", "shared-cache")
	require.NotNil(t, cache)

	loadDone := make(chan struct{})
	go func() {
		defer close(loadDone)
		_, _ = cache.Load(ctx, "memo-key")
	}()

	select {
	case <-queries.loadStart:
	case <-time.After(time.Second):
		t.Fatal("expected SQL load to start")
	}

	setQueriesDone := make(chan struct{})
	go func() {
		factory.SetQueries(nil)
		close(setQueriesDone)
	}()

	select {
	case <-setQueriesDone:
		t.Fatal("expected SetQueries to wait for in-flight SQL load")
	case <-time.After(50 * time.Millisecond):
	}

	close(queries.loadBlock)

	select {
	case <-setQueriesDone:
	case <-time.After(time.Second):
		t.Fatal("expected SetQueries to finish after load completes")
	}

	select {
	case <-loadDone:
	case <-time.After(time.Second):
		t.Fatal("expected SQL load goroutine to finish")
	}
}
