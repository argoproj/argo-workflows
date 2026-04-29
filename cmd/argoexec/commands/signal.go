package commands

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/argoproj/argo-workflows/v4/util/file"
	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// startFileSignalHandler starts a goroutine that watches a signal file via
// inotify. Whenever the file is written to, the integer signal value is read,
// the file is removed, and the signal is forwarded to the given process.
func startFileSignalHandler(ctx context.Context, pid int) {
	logger := logging.RequireLoggerFromContext(ctx)
	signalPath := filepath.Clean(filepath.Join(varRunArgo, "ctr", containerName, "signal"))
	logger.WithField("signalPath", signalPath).Info(ctx, "waiting for signals")

	go func() {
		err := file.WatchFile(ctx, signalPath, func() {
			data, readErr := os.ReadFile(signalPath)
			if readErr != nil {
				return
			}
			s, parseErr := strconv.Atoi(strings.TrimSpace(string(data)))
			if parseErr != nil || s <= 0 {
				return
			}
			_ = os.Remove(signalPath)
			logger.WithFields(logging.Fields{
				"signal":     s,
				"signalPath": signalPath,
			}).Info(ctx, "received signal")
			_ = osspecific.Kill(pid, syscall.Signal(s))
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.WithError(err).Info(ctx, "file signal handler exited")
			return
		}
		logger.Info(ctx, "file signal handler exiting due to context cancellation")
	}()
}
