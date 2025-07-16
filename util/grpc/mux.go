package grpc

import (
	"net/http"
	"net/textproto"
	"strings"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func IncomingHeaderMatcher(key string) (string, bool) {
	switch textproto.CanonicalMIMEHeaderKey(key) {
	case
		// Don't forward Content-Length as that will lead to "stream terminated
		// by RST_STREAM with error code: PROTOCOL_ERROR" errors for requests with a body.
		// Reference: https://github.com/grpc-ecosystem/grpc-gateway/issues/2682#issuecomment-1125470811
		"Content-Length",

		// Don't forward connection-specific headers.
		// "An endpoint MUST NOT generate an HTTP/2 message containing
		// connection-specific header fields. This includes the Connection
		// header field and those listed as having connection-specific semantics
		// in Section 7.6.1 of [HTTP] (that is, Proxy-Connection, Keep-Alive,
		// Transfer-Encoding, and Upgrade)."
		// Reference: https://httpwg.org/specs/rfc9113.html#ConnectionSpecific
		"Connection",
		"Keep-Alive",
		"Proxy-Connection",
		"Transfer-Encoding",
		"Upgrade":
		return key, false

	default:
		return key, true
	}
}

// NewMuxHandler returns an HTTP handler that allows serving both gRPC and
// HTTP requests over the same port, both with and without TLS enabled.
// From: https://pkg.go.dev/golang.org/x/net@v0.41.0/http2/h2c#NewHandler
// "If a request is an h2c connection, it's hijacked and redirected to
// s.ServeConn. Otherwise the returned Handler just forwards requests to h. This
// works because h2c is designed to be parseable as valid HTTP/1, but ignored by
// any HTTP server that does not handle h2c. Therefore we leverage the HTTP/1
// compatible parts of the Go http library to parse and recognize h2c requests.
// Once a request is recognized as h2c, we hijack the connection and convert it
// to an HTTP/2 connection which is understandable to s.ServeConn. (s.ServeConn
// understands HTTP/2 except for the h2c part of it.)"
func NewMuxHandler(grpcServerHandler http.Handler, httpServerHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Match against "Content-Type", which is guaranteed to start with "application/grpc" for gRPC requests.
		// Spec: https://chromium.googlesource.com/external/github.com/grpc/grpc/+/HEAD/doc/PROTOCOL-HTTP2.md
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServerHandler.ServeHTTP(w, r)
		} else {
			httpServerHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
