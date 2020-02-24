package apiclient

import (
	"context"
)

type abstractIntermediary struct {
	panicIntermediary
	ctx    context.Context
	cancel context.CancelFunc
	error  chan error
}

func (w watchIntermediary) Context() context.Context {
	return w.ctx
}

func newAbstractIntermediary(ctx context.Context) abstractIntermediary {
	ctx, cancel := context.WithCancel(ctx)
	return abstractIntermediary{
		panicIntermediary: panicIntermediary{},
		ctx:               ctx,
		cancel:            cancel,
		error:             make(chan error, 1),
	}
}
