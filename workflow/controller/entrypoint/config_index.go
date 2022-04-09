package entrypoint

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/config"
)

type configIndex map[string]config.Image

func (c configIndex) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	v, ok := c[image]
	if !ok {
		return nil, nil
	}
	return &Image{Command: v.Command, Args: v.Args}, nil
}

var _ Interface = &configIndex{}
