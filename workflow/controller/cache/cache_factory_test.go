package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	memodb "github.com/argoproj/argo-workflows/v4/util/memo/db"
)

func TestCacheFactoryUsesControllerNamespace(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	client := fake.NewSimpleClientset()
	factory := NewCacheFactory(client, "controller")
	cache := factory.GetCache("shared-cache")
	assert.Same(t, cache, factory.GetCache("shared-cache"))
	require.NoError(t, cache.Save(ctx, "key", "node", &wfv1.Outputs{}, ""))
	_, err := client.CoreV1().ConfigMaps("controller").Get(ctx, "shared-cache", metav1.GetOptions{})
	require.NoError(t, err)
	queries := &testMemoizationDB{enabled: true}
	factory.SetQueries(ctx, queries)
	sqlCache := factory.GetCache("shared-cache")
	require.NoError(t, sqlCache.Save(ctx, "key", "node", &wfv1.Outputs{}, "1h"))
	assert.Equal(t, "controller", queries.namespace)
	assert.Equal(t, "shared-cache", queries.cacheName)
	assert.NotSame(t, cache, sqlCache)
}

type testMemoizationDB struct {
	enabled   bool
	saveCalls atomic.Int32
	start     chan struct{}
	block     chan struct{}
	closed    chan struct{}
	namespace string
	cacheName string
}

func (q *testMemoizationDB) wait() {
	if q.start != nil {
		close(q.start)
	}
	if q.block != nil {
		<-q.block
	}
}
func (q *testMemoizationDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	q.wait()
	return nil, nil
}
func (q *testMemoizationDB) Save(_ context.Context, ns, name, _, _ string, _ *wfv1.Outputs, _ string) error {
	q.wait()
	q.namespace, q.cacheName = ns, name
	q.saveCalls.Add(1)
	return nil
}
func (q *testMemoizationDB) Prune(context.Context) (int64, error) { q.wait(); return 0, nil }
func (q *testMemoizationDB) IsEnabled() bool                      { return q.enabled }
func (q *testMemoizationDB) Close() error {
	if q.closed != nil {
		close(q.closed)
	}
	return nil
}

func TestCacheFactoryStaleSQLCacheNoopsAfterDisable(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset(), "controller")
	queries := &testMemoizationDB{enabled: true}
	factory.SetQueries(ctx, queries)
	cache := factory.GetCache("shared-cache")
	factory.SetQueries(ctx, nil)
	require.NoError(t, cache.Save(ctx, "key", "node", nil, "1h"))
	entry, err := cache.Load(ctx, "key")
	require.NoError(t, err)
	assert.Nil(t, entry)
	assert.Zero(t, queries.saveCalls.Load())
}

func TestCacheFactoryReconfigureDrainsQueriesWithoutHoldingLock(t *testing.T) {
	for _, operation := range []string{"load", "save", "prune"} {
		t.Run(operation, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			factory := NewCacheFactory(fake.NewSimpleClientset(), "controller")
			old := &testMemoizationDB{enabled: true, start: make(chan struct{}), block: make(chan struct{}), closed: make(chan struct{})}
			factory.SetQueries(ctx, old)
			cache := factory.GetCache("shared-cache")
			release := sync.OnceFunc(func() { close(old.block) })
			t.Cleanup(release)
			done := make(chan error, 1)
			go func() {
				switch operation {
				case "save":
					done <- cache.Save(ctx, "key", "node", nil, "1h")
				case "load":
					_, err := cache.Load(ctx, "key")
					done <- err
				case "prune":
					done <- factory.WithQueries(func(q memodb.MemoizationDB) error { _, err := q.Prune(ctx); return err })
				}
			}()
			select {
			case <-old.start:
			case <-time.After(time.Second):
				t.Fatal("query did not start")
			}
			replacement := &testMemoizationDB{enabled: true}
			reconfigured := make(chan struct{})
			go func() { factory.SetQueries(ctx, replacement); close(reconfigured) }()
			select {
			case <-reconfigured:
			case <-time.After(time.Second):
				t.Fatal("configuration waited for query I/O")
			}
			select {
			case <-old.closed:
				t.Fatal("closed a backend with an in-flight query")
			default:
			}
			require.NoError(t, factory.GetCache("shared-cache").Save(ctx, "new-key", "new-node", nil, ""))
			assert.EqualValues(t, 1, replacement.saveCalls.Load())
			release()
			require.NoError(t, <-done)
			select {
			case <-old.closed:
			case <-time.After(time.Second):
				t.Fatal("retired backend was not closed after draining")
			}
		})
	}
}
