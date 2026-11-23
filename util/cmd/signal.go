package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// ShutdownContext returns a copy of ctx that is cancelled when the process
// receives SIGTERM (as sent by the kubelet when it stops a pod) or SIGINT.
// Calling stop cancels the context and restores the default signal behaviour,
// so a second signal kills the process.
func ShutdownContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
}
