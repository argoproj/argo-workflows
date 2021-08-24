package pprof

import (
	"net/http"
	"net/http/pprof"
	"os"

	log "github.com/sirupsen/logrus"
)

func Init() {
	// https://mmcloughlin.com/posts/your-pprof-is-showing
	http.DefaultServeMux = http.NewServeMux()
	if os.Getenv("ARGO_PPROF") == "true" {
		log.Info("enabling pprof debug endpoints - do not do this in production")
		http.HandleFunc("/debug/pprof/", pprof.Index)
		http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		http.HandleFunc("/debug/pprof/profile", pprof.Profile)
		http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		http.HandleFunc("/debug/pprof/trace", pprof.Trace)
	} else {
		log.Info("not enabling pprof debug endpoints")
	}
}
