package entrypoint

import (
	"context"

	"k8s.io/utils/lru"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type cacheIndex struct {
	cache    *lru.Cache
	delegate Interface
}

func (i *cacheIndex) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	logger := logging.RequireLoggerFromContext(ctx)
	if cmd, ok := i.cache.Get(image); ok {
		logger.WithFields(logging.Fields{
			"image": image,
			"cmd":   cmd,
		}).Debug(ctx, "Cache hit")
		return cmd.(*Image), nil
	}
	logger.WithField("image", image).Debug(ctx, "Cache miss")
	v, err := i.delegate.Lookup(ctx, image, options)
	if err != nil {
		return nil, err
	}
	i.cache.Add(image, v)
	return v, nil
}
