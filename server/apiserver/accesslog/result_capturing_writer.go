package accesslog

import (
	"net/http"
)

// resultCapturingWriter captures the size and status code of the response.
// Because http.response implements http.Flusher, we must do so too, otherwise Watch* methods don't work.
// We do not implement http.Hijacker, as HTTP/2 requests should not allow it.
type resultCapturingWriter struct {
	http.ResponseWriter // MUST also be http.Flusher
	status              int
	size                int
}

func (r *resultCapturingWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.size += size
	return size, err
}

func (r *resultCapturingWriter) WriteHeader(v int) {
	r.ResponseWriter.WriteHeader(v)
	r.status = v
}

func (r *resultCapturingWriter) Flush() {
	r.ResponseWriter.(http.Flusher).Flush()
}
