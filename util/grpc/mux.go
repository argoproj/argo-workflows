package grpc

import (
	"net/http"
	"net/textproto"
	"strings"
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

// SanitizeCookieHeader strips cookie pairs containing non-printable ASCII
// characters from the request's Cookie headers.
//
// Browsers send every cookie stored for the server's host, including cookies
// set by unrelated applications. The grpc-gateway forwards the Cookie header
// as gRPC metadata, and the gRPC transport rejects values containing
// non-printable ASCII characters, failing every API request with an Internal
// error (e.g. `header key "cookie" contains value with non-printable ASCII
// characters`). Dropping only the offending pairs keeps valid cookies —
// including the Argo authorization cookie, which is always printable ASCII —
// intact.
func SanitizeCookieHeader(r *http.Request) {
	values := r.Header.Values("Cookie")
	if len(values) == 0 {
		return
	}
	sanitized := make([]string, 0, len(values))
	for _, v := range values {
		if isPrintableASCII(v) {
			sanitized = append(sanitized, v)
			continue
		}
		var kept []string
		for _, pair := range strings.Split(v, ";") {
			pair = strings.TrimLeft(pair, " ")
			if pair == "" || !isPrintableASCII(pair) {
				continue
			}
			kept = append(kept, pair)
		}
		if len(kept) > 0 {
			sanitized = append(sanitized, strings.Join(kept, "; "))
		}
	}
	r.Header.Del("Cookie")
	for _, v := range sanitized {
		r.Header.Add("Cookie", v)
	}
}

func isPrintableASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// NewMuxHandler returns an HTTP handler that allows serving both gRPC and
// HTTP requests over the same port, both with and without TLS enabled.
// It dispatches HTTP/2 requests with a gRPC content type to the gRPC handler
// and forwards all other requests to the HTTP handler. Unencrypted HTTP/2 (h2c)
// support is enabled by the serving [http.Server] via its Protocols field; see
// the call site in the Argo server.
func NewMuxHandler(grpcServerHandler http.Handler, httpServerHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Match against "Content-Type", which is guaranteed to start with "application/grpc" for gRPC requests.
		// Spec: https://chromium.googlesource.com/external/github.com/grpc/grpc/+/HEAD/doc/PROTOCOL-HTTP2.md
		if r.ProtoMajor == 2 && strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServerHandler.ServeHTTP(w, r)
		} else {
			httpServerHandler.ServeHTTP(w, r)
		}
	})
}
