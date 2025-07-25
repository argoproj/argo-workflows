package accesslog

import (
	"net/http"
	"time"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type LoggingInterceptor struct {
	logger logging.Logger
}

func NewLoggingInterceptor(logger logging.Logger) *LoggingInterceptor {
	return &LoggingInterceptor{logger: logger}
}

// Interceptor returns a handler that provides access logging.
//
// github.com/gorilla/handlers/logging.go
// https://arunvelsriram.medium.com/simple-golang-http-logging-middleware-315656ff8722
func (i *LoggingInterceptor) Interceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		rcw := &resultCapturingWriter{ResponseWriter: w}

		h.ServeHTTP(rcw, r)

		i.logger.WithFields(logging.Fields{
			"path":     r.URL.Path, // log the path not the URL, to avoid logging sensitive data that could be in the query params
			"method":   r.Method,   // log the method, so we can differentiate create/update from get/list
			"status":   rcw.status,
			"size":     rcw.size,
			"duration": time.Since(t),
		}).Info(r.Context(), "HTTP request")
	})
}
