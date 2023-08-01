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
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	err := commands.NewRootCommand().ExecuteContext(ctx)
	if err != nil {
		println(err.Error())
		util.WriteTerminateMessage(err.Error())
		if exitError, ok := err.(errors.Exited); ok {
			if exitError.ExitCode() >= 0 {
				os.Exit(exitError.ExitCode())
			} else {
				os.Exit(137) // probably SIGTERM or SIGKILL
			}
		} else {
			os.Exit(64)
		}
	}
}
