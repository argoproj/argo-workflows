package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/argoproj/argo-workflows/v3/util/errors"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/commands"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(logging.WithLogger(context.Background(), logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())), syscall.SIGTERM)
	defer stop()
	err := commands.NewRootCommand().ExecuteContext(ctx)
	if err != nil {
		if exitError, ok := err.(errors.Exited); ok {
			if exitError.ExitCode() >= 0 {
				os.Exit(exitError.ExitCode())
			} else {
				os.Exit(137) // probably SIGTERM or SIGKILL
			}
		} else {
			util.WriteTerminateMessage(err.Error()) // we don't want to overwrite any other message
			println(err.Error())
			os.Exit(64)
		}
	}
}
