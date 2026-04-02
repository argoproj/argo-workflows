package commands

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
	"github.com/argoproj/argo-workflows/v4/workflow/executor/osspecific"
)

// startFileSignalHandler starts a goroutine that polls for signals written to a file.
// It reads the signal file every 2 seconds and forwards signals to the given process.
func startFileSignalHandler(ctx context.Context, pid int) {
	logger := logging.RequireLoggerFromContext(ctx)
	signalPath := filepath.Clean(filepath.Join(varRunArgo, "ctr", containerName, "signal"))
	logger.WithField("signalPath", signalPath).Info(ctx, "waiting for signals")

	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info(ctx, "file signal handler exiting due to context cancellation")
				return
			case <-ticker.C:
				data, err := os.ReadFile(signalPath)
				if err != nil {
					continue
				}
				s, err := strconv.Atoi(string(data))
				if err != nil || s <= 0 {
					continue
				}
				_ = os.Remove(signalPath)
				logger.WithFields(logging.Fields{
					"signal":     s,
					"signalPath": signalPath,
				}).Info(ctx, "received signal")
				_ = osspecific.Kill(pid, syscall.Signal(s))
			}
		}
	}()
}
