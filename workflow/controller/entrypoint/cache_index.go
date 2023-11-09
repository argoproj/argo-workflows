package entrypoint

import (
	"context"

	log "github.com/sirupsen/logrus"
	"k8s.io/utils/lru"
)

type cacheIndex struct {
	cache    *lru.Cache
	delegate Interface
}

func (i *cacheIndex) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	if cmd, ok := i.cache.Get(image); ok {
		log.WithField("image", image).WithField("cmd", cmd).Debug("Cache hit")
		return cmd.(*Image), nil
	}
	log.WithField("image", image).Debug("Cache miss")
	v, err := i.delegate.Lookup(ctx, image, options)
	if err != nil {
		return nil, err
	}
	i.cache.Add(image, v)
	return v, nil
}
