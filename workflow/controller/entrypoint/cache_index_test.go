package entrypoint

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/utils/lru"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type lookupFunc func(ctx context.Context, image string, options Options) (*Image, error)

func (f lookupFunc) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	return f(ctx, image, options)
}

func TestCacheIndexLookupAlwaysRefreshesCache(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	first := &Image{Entrypoint: []string{"/first"}}
	refreshed := &Image{Entrypoint: []string{"/refreshed"}}
	lookups := 0
	index := &cacheIndex{
		cache: lru.New(1),
		delegate: lookupFunc(func(context.Context, string, Options) (*Image, error) {
			lookups++
			if lookups == 1 {
				return first, nil
			}
			return refreshed, nil
		}),
	}

	actual, err := index.Lookup(ctx, "example.com/app:latest", Options{ImagePullPolicy: apiv1.PullAlways})
	require.NoError(t, err)
	assert.Same(t, first, actual)

	actual, err = index.Lookup(ctx, "example.com/app:latest", Options{ImagePullPolicy: apiv1.PullAlways})
	require.NoError(t, err)
	assert.Same(t, refreshed, actual)

	actual, err = index.Lookup(ctx, "example.com/app:latest", Options{ImagePullPolicy: apiv1.PullIfNotPresent})
	require.NoError(t, err)
	assert.Same(t, refreshed, actual)
	assert.Equal(t, 2, lookups)
}

func TestCacheIndexLookupUsesCacheForOrdinaryPullPolicies(t *testing.T) {
	tests := []struct {
		name   string
		policy apiv1.PullPolicy
	}{
		{name: "empty", policy: ""},
		{name: "if-not-present", policy: apiv1.PullIfNotPresent},
		{name: "never", policy: apiv1.PullNever},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := logging.TestContext(t.Context())
			expected := &Image{Entrypoint: []string{"/entrypoint"}}
			lookups := 0
			index := &cacheIndex{
				cache: lru.New(1),
				delegate: lookupFunc(func(context.Context, string, Options) (*Image, error) {
					lookups++
					return expected, nil
				}),
			}

			for range 2 {
				actual, err := index.Lookup(ctx, "example.com/app:stable", Options{ImagePullPolicy: test.policy})
				require.NoError(t, err)
				assert.Same(t, expected, actual)
			}
			assert.Equal(t, 1, lookups)
		})
	}
}

func TestCacheIndexLookupAlwaysDoesNotFallBackToStaleValue(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	stale := &Image{Entrypoint: []string{"/stale"}}
	wantErr := errors.New("registry lookup failed")
	lookups := 0
	index := &cacheIndex{
		cache: lru.New(1),
		delegate: lookupFunc(func(context.Context, string, Options) (*Image, error) {
			lookups++
			if lookups == 1 {
				return stale, nil
			}
			return nil, wantErr
		}),
	}

	actual, err := index.Lookup(ctx, "example.com/app:latest", Options{})
	require.NoError(t, err)
	assert.Same(t, stale, actual)

	actual, err = index.Lookup(ctx, "example.com/app:latest", Options{ImagePullPolicy: apiv1.PullAlways})
	require.ErrorIs(t, err, wantErr)
	assert.Nil(t, actual)

	actual, err = index.Lookup(ctx, "example.com/app:latest", Options{})
	require.NoError(t, err)
	assert.Same(t, stale, actual)
	assert.Equal(t, 2, lookups)
}
