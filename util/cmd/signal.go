package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ShutdownContext returns a copy of ctx that is cancelled when the process
// receives SIGTERM (as sent by the kubelet when it stops a pod) or SIGINT.
// The default signal behaviour is restored as soon as the context is done,
// so a second signal kills the process even while shutdown is still running.
func ShutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	context.AfterFunc(ctx, stop)
	return ctx, stop
}
