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
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer stop()
	err := commands.NewRootCommand().ExecuteContext(ctx)
	if err != nil {
		if exitError, ok := err.(errors.Exited); ok {
			if exitError.ExitCode() >= 0 {
				return exitError.ExitCode()
			}
			return 137 // probably SIGTERM or SIGKILL
		}
		util.WriteTerminateMessage(err.Error()) // we don't want to overwrite any other message
		println(err.Error())
		return 64
	}
	return 0
}
