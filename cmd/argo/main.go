package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := commands.NewCommand().ExecuteContext(ctx)
	stop()
	if err != nil {
		os.Exit(1)
	}
}
