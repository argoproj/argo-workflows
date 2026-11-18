package cache

import (
	"context"
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
	factory.SetQueries(queries)
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
	namespace string
	cacheName string
}

func (t *testMemoizationDB) wait() {
	if t.start != nil {
		close(t.start)
	}
	if t.block != nil {
		<-t.block
	}
}
func (t *testMemoizationDB) Load(context.Context, string, string, string) (*memodb.CacheRecord, error) {
	t.wait()
	return nil, nil
}
func (t *testMemoizationDB) Save(_ context.Context, namespace, cacheName, _, _ string, _ *wfv1.Outputs, _ string) error {
	t.wait()
	t.namespace, t.cacheName = namespace, cacheName
	t.saveCalls.Add(1)
	return nil
}
func (*testMemoizationDB) Prune(context.Context) (int64, error) { return 0, nil }
func (t *testMemoizationDB) IsEnabled() bool                    { return t.enabled }

func TestCacheFactoryStaleSQLCacheNoopsAfterDisable(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	factory := NewCacheFactory(fake.NewSimpleClientset(), "controller")
	queries := &testMemoizationDB{enabled: true}
	factory.SetQueries(queries)
	cache := factory.GetCache("shared-cache")
	factory.SetQueries(nil)
	require.NoError(t, cache.Save(ctx, "key", "node", &wfv1.Outputs{}, "1h"))
	assert.Zero(t, queries.saveCalls.Load())
}

func TestCacheFactoryReconfigureDoesNotWaitForQueries(t *testing.T) {
	for _, operation := range []string{"load", "save"} {
		t.Run(operation, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			factory := NewCacheFactory(fake.NewSimpleClientset(), "controller")
			queries := &testMemoizationDB{enabled: true, start: make(chan struct{}), block: make(chan struct{})}
			factory.SetQueries(queries)
			cache := factory.GetCache("shared-cache")
			done := make(chan error, 1)
			go func() {
				if operation == "save" {
					done <- cache.Save(ctx, "key", "node", &wfv1.Outputs{}, "1h")
				} else {
					_, err := cache.Load(ctx, "key")
					done <- err
				}
			}()
			t.Cleanup(func() { close(queries.block); require.NoError(t, <-done) })
			select {
			case <-queries.start:
			case <-time.After(time.Second):
				t.Fatal("query did not start")
			}
			reconfigured := make(chan struct{})
			go func() { factory.SetQueries(nil); close(reconfigured) }()
			select {
			case <-reconfigured:
			case <-time.After(time.Second):
				t.Fatal("factory lock held during SQL query")
			}
		})
	}
}
