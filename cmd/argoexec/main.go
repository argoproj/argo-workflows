package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/argoproj/argo-workflows/v4/util/errors"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/commands"
	"github.com/argoproj/argo-workflows/v4/util"
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
			// An Exited error that wraps a cause (e.g. the init-less supervisor
			// pre-main failure) carries a descriptive message worth surfacing as the
			// termination message; a bare exit-status error (the user command's own
			// exit) wraps nothing and reports only its code. The cause is already
			// logged at its source.
			if _, ok := err.(interface{ Unwrap() error }); ok {
				util.WriteTerminateMessage(err.Error())
			}
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
