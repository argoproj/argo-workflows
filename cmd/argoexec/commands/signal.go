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

// forwardSignals starts a goroutine that forwards OS signals from the given
// channel to the process with the given pid. Forwarding stops when the channel
// closes or when the caller calls the returned stop function. Cancellation of
// the parent ctx must not stop forwarding, because signal.NotifyContext can
// cancel ctx before signal.Notify sends the same SIGTERM to signals. Stop waits
// for the goroutine to exit. Call stop before the next attempt starts. The
// goroutine drops signals that can be ignored. When ignoreTerm is true, it also
// drops SIGTERM: artifact sidecars stay alive to help the aux container, and
// file signals stop them. The caller owns the lifecycle of the channel
// (signal.Notify / signal.Reset / close).
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
