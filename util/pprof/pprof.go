package pprof

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Enabled reports whether the pprof debug endpoints are turned on.
func Enabled() bool {
	return os.Getenv("ARGO_PPROF") == "true"
}

func Init(ctx context.Context) {
	// https://mmcloughlin.com/posts/your-pprof-is-showing
	http.DefaultServeMux = http.NewServeMux()
	logger := logging.RequireLoggerFromContext(ctx)
	if Enabled() {
		logger.Info(ctx, "enabling pprof debug endpoints - do not do this in production")
		http.HandleFunc("/debug/pprof/", pprof.Index)
		http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		http.HandleFunc("/debug/pprof/profile", pprof.Profile)
		http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		http.HandleFunc("/debug/pprof/trace", pprof.Trace)
	} else {
		logger.Info(ctx, "not enabling pprof debug endpoints")
	}
}

// ListenLoopback serves the endpoints registered by Init on the loopback
// interface, and does nothing when they are disabled. It returns once the
// listener is up, leaving the server running in the background until ctx is
// done. It reports the address it bound, or "" when the endpoints are off or
// the port could not be taken.
//
// Loopback rather than all interfaces, because unlike the controller and the
// server, this runs inside a user's workflow pod: anything bound to the pod IP
// is reachable from the cluster network, and these endpoints are unauthenticated
// and can dump memory. `kubectl port-forward` reaches a loopback listener
// inside the pod, so profiles are still retrievable.
//
// The port is configurable because this process shares a network namespace
// with the user's own containers, which may already be using the default.
func ListenLoopback(ctx context.Context, port string) string {
	logger := logging.RequireLoggerFromContext(ctx)
	if !Enabled() {
		return ""
	}
	addr := net.JoinHostPort("127.0.0.1", port)
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		// Profiling is a debugging aid: losing the port must never take the
		// process down with it.
		logger.WithError(err).WithField("addr", addr).Warn(ctx, "could not start pprof listener")
		return ""
	}
	srv := &http.Server{Handler: http.DefaultServeMux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		if err := srv.Close(); err != nil {
			logger.WithError(err).Warn(ctx, "error closing pprof listener")
		}
	}()
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Warn(ctx, "pprof listener stopped")
		}
	}()
	bound := ln.Addr().String()
	logger.WithField("addr", bound).Info(ctx, "pprof debug endpoints listening")
	return bound
}
