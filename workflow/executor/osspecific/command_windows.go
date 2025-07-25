package osspecific

import (
	"context"
	"os"
	"os/exec"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func StartCommand(ctx context.Context, cmd *exec.Cmd) (func(), error) {
	logger := logging.RequireLoggerFromContext(ctx)
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}

	if isTerminal(cmd.Stdin) {
		logger.Warn(ctx, "TTY detected but is not supported on windows")
	}
	return simpleStart(cmd)
}

func simpleStart(cmd *exec.Cmd) (func(), error) {
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	closer := func() {
	}

	return closer, nil
}
