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
// given channel to the process with the given pid, until the channel is closed
// or the returned stop function is called. Parent cancellation must not stop
// forwarding: signal.NotifyContext can cancel ctx before signal.Notify delivers
// the same SIGTERM to signals. Stop waits for the goroutine to exit and must be
// called before starting another attempt. Signals that can be ignored are
// dropped. When ignoreTerm is true, SIGTERM is also dropped: artifact sidecars
// stay alive to assist the aux container and are terminated via file signals.
// The caller owns the channel's lifecycle (signal.Notify / signal.Reset / close).
func forwardSignals(ctx context.Context, signals <-chan os.Signal, pid int, ignoreTerm bool) func() {
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	done := make(chan struct{})
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
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			case s, ok := <-signals:
				if !ok {
					return
				}
				forward(s)
			}
		}
	}()
	return func() {
		cancel()
		<-done
	}
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
