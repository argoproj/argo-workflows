package resource

import (
	"context"

	"k8s.io/utils/lru"
)

// cache provides a LRU cache, only suitable for short lived use cases, because the cache does not have time expiry.
type cache struct {
	cache    *lru.Cache
	delegate Interface
}

func (r *cache) GetSecret(ctx context.Context, name, key string) (string, error) {
	k := "secret/" + name + "/"
	v, ok := r.cache.Get(k)
	if ok {
		return v.(string), nil
	}
	s, err := r.delegate.GetSecret(ctx, name, key)
	if err != nil {
		return "", err
	}
	r.cache.Add(k, s)
	return s, nil
}

func (r *cache) GetConfigMapKey(ctx context.Context, name, key string) (string, error) {
	k := "configmap/" + name + "/"
	v, ok := r.cache.Get(k)
	if ok {
		return v.(string), nil
	}
	s, err := r.delegate.GetConfigMapKey(ctx, name, key)
	if err != nil {
		return "", err
	}
	r.cache.Add(k, s)
	return s, nil
}
