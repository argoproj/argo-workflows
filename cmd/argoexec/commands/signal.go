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

// forwardSignals starts a goroutine that forwards OS signals received on the
// given channel to the process with the given pid, until the channel is
// closed or ctx is done. Signals that can be ignored are dropped; when
// ignoreTerm is true SIGTERM is dropped as well (artifact sidecars stay alive
// to assist the aux container and are terminated only via the file-signal
// mechanism). The caller owns the channel's lifecycle (signal.Notify /
// signal.Reset / close). A caller that runs the process more than once
// (the emissary's retry loop) scopes ctx to the attempt, so signals never go
// to a pid from an earlier attempt.
func forwardSignals(ctx context.Context, signals <-chan os.Signal, pid int, ignoreTerm bool) {
	logger := logging.RequireLoggerFromContext(ctx)
	forward := func(s os.Signal) {
		if osspecific.CanIgnoreSignal(s) || (ignoreTerm && s == syscall.SIGTERM) {
			logger.WithField("signal", s).Debug(ctx, "ignore signal")
			return
		}
		logger.WithField("signal", s).Debug(ctx, "forwarding signal")
		_ = osspecific.Kill(pid, s.(syscall.Signal))
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				// The same SIGTERM that cancels ctx (main.go's NotifyContext)
				// may already be buffered here; deliver it before leaving,
				// as the pre-ctx forwarder always did.
				for {
					select {
					case s, ok := <-signals:
						if !ok {
							return
						}
						forward(s)
					default:
						return
					}
				}
			case s, ok := <-signals:
				if !ok {
					return
				}
				forward(s)
			}
		}
	}()
}

// startFileSignalHandler starts a goroutine that watches a signal file via
// inotify. Whenever the file is written to, the integer signal value is read,
// the file is removed, and the signal is forwarded to the given process.
func startFileSignalHandler(ctx context.Context, pid int, containerName string) {
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
