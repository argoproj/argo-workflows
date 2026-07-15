package pprof

import (
	"context"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func Init(ctx context.Context) {
	// https://mmcloughlin.com/posts/your-pprof-is-showing
	http.DefaultServeMux = http.NewServeMux()
	logger := logging.RequireLoggerFromContext(ctx)
	if os.Getenv("ARGO_PPROF") == "true" {
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
