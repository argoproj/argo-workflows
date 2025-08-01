package accesslog

import (
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// Interceptor returns a handler that provides access logging.
//
// github.com/gorilla/handlers/logging.go
// https://arunvelsriram.medium.com/simple-golang-http-logging-middleware-315656ff8722
func Interceptor(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()

		rcw := &resultCapturingWriter{ResponseWriter: w}

		h.ServeHTTP(rcw, r)

		log.WithFields(log.Fields{
			"path":     r.URL.Path, // log the path not the URL, to avoid logging sensitive data that could be in the query params
			"method":   r.Method,   // log the method, so we can differentiate create/update from get/list
			"status":   rcw.status,
			"size":     rcw.size,
			"duration": time.Since(t),
		}).Info()
	})
}
