package apiclient

import (
	"context"
)

type abstractIntermediary struct {
	panicIntermediary
	// nolint: containedctx
	ctx    context.Context
	cancel context.CancelFunc
	// if anything is on this channel, then then we must be done - the error maybe io.EOF - which just means stop
	error chan error
}

func (w abstractIntermediary) Context() context.Context {
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
