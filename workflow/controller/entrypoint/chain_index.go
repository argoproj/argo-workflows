package entrypoint

import (
	"context"
	"fmt"
)

type chainIndex []Interface

func (c chainIndex) Lookup(ctx context.Context, image string, options Options) (*Image, error) {
	for _, i := range c {
		v, err := i.Lookup(ctx, image, options)
		if v != nil || err != nil {
			return v, err
		}
	}
	return nil, fmt.Errorf("image not found")
}

var _ Interface = chainIndex{}
