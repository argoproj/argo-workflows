package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/util/logging"
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
