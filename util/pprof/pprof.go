package pprof

import (
	"context"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// Init resets the default mux and returns the profiling handler when enabled.
func Init(ctx context.Context) *http.ServeMux {
	// https://mmcloughlin.com/posts/your-pprof-is-showing
	mux := http.NewServeMux()
	http.DefaultServeMux = mux
	logger := logging.RequireLoggerFromContext(ctx)
	if os.Getenv("ARGO_PPROF") == "true" {
		logger.Info(ctx, "enabling pprof debug endpoints - do not do this in production")
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		return mux
	}
	logger.Info(ctx, "not enabling pprof debug endpoints")
	return nil
}
