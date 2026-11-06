// Package gateway holds the grpc-gateway v2 glue that both the server and the
// generated pkg/apiclient code depend on. It must stay a leaf package: no
// server-side dependencies (interceptors, rate limiters), because pkg/apiclient
// is the public Go SDK surface and imports it.
package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/proto"

	"github.com/argoproj/argo-workflows/v4/util/fields"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

type messageMarshaler struct {
	cleaner fields.Cleaner
	isSSE   bool
}

var errMarshalOnly = errors.New("stream marshaler only supports Marshal")

// grpc-gateway's ForwardResponseStream only ever calls Marshal on this
// marshaler; the rest of the runtime.Marshaler surface fails loudly so a
// gateway upgrade that starts using it cannot silently misbehave.
func (m *messageMarshaler) Unmarshal(data []byte, v any) error { return errMarshalOnly }
func (m *messageMarshaler) NewDecoder(r io.Reader) runtime.Decoder {
	return runtime.DecoderFunc(func(v any) error { return errMarshalOnly })
}
func (m *messageMarshaler) NewEncoder(w io.Writer) runtime.Encoder {
	return runtime.EncoderFunc(func(v any) error { return errMarshalOnly })
}

func (m *messageMarshaler) ContentType(_ any) string {
	if m.isSSE {
		return "text/event-stream"
	}
	return "application/json"
}

func (m *messageMarshaler) Marshal(v any) ([]byte, error) {
	// grpc-gateway wraps every stream message in map[string]any{"result": msg}
	// (or {"error": ...}), so v is always a JSON object. Cleaner round-trips it
	// through JSON to apply the ?fields filter.
	out := v
	var cleaned map[string]any
	if changed, err := m.cleaner.Clean(v, &cleaned); err != nil {
		return nil, err
	} else if changed {
		out = cleaned
	}
	dataBytes, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	if m.isSSE {
		dataBytes = fmt.Appendf(nil, "data: %s \n\n", string(dataBytes))
	}
	return dataBytes, nil
}

func newStreamMarshaler(req *http.Request, isSSE bool) *messageMarshaler {
	return &messageMarshaler{
		cleaner: fields.NewCleaner(req.URL.Query().Get("fields")),
		isSSE:   isSSE,
	}
}

// fallbackLogger is used when a request context carries no logger. This package
// is wired into generated pkg/apiclient code, so it can run under HTTP servers
// (or SDK consumers) that never call logging.WithLogger — and the keepalive runs
// in a background goroutine, where a missing-logger panic would kill the whole
// process on something as routine as an SSE client disconnecting.
var fallbackLogger = logging.NewSlogLogger(logging.Info, logging.Text)

func loggerFromContext(ctx context.Context) logging.Logger {
	if logger := logging.GetLoggerFromContextOrNil(ctx); logger != nil {
		return logger
	}
	return fallbackLogger
}

// flush recovers from panics because it runs in a background goroutine.
func flush(ctx context.Context, flusher http.Flusher) {
	defer func() {
		if r := recover(); r != nil {
			loggerFromContext(ctx).Warn(ctx, "recovered in flush, issue with writer inside http.ResponseWriter")
		}
	}()
	flusher.Flush()
}

func writeKeepalive(ctx context.Context, w http.ResponseWriter, mut *sync.Mutex) bool {
	mut.Lock()
	defer mut.Unlock()

	_, err := w.Write([]byte(":\n"))
	if err != nil {
		loggerFromContext(ctx).WithError(err).Warn(ctx, "failed to write http keepalive response")
		return false
	}
	if f, ok := w.(http.Flusher); ok {
		flush(ctx, f)
	}
	return true
}

// keepaliveInterval is a variable so tests can shorten it.
var keepaliveInterval = 15 * time.Second

func keepalive(ctx context.Context, w http.ResponseWriter, mut *sync.Mutex) {
	keepaliveTicker := time.NewTicker(keepaliveInterval)
	defer keepaliveTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-keepaliveTicker.C:
			if !writeKeepalive(ctx, w, mut) {
				return
			}
		}
	}
}

// mutexWriter wraps an http.ResponseWriter with a mutex to prevent
// concurrent writes from the keepalive goroutine and the main handler.
type mutexWriter struct {
	http.ResponseWriter
	mut *sync.Mutex
}

func (w *mutexWriter) Write(p []byte) (int, error) {
	w.mut.Lock()
	defer w.mut.Unlock()
	return w.ResponseWriter.Write(p)
}

// Flush makes the wrapper usable with http.NewResponseController, which
// grpc-gateway's ForwardResponseStream uses to flush after every message.
// Without it the controller reports ErrNotSupported and the gateway aborts the
// stream with an HTTP 500. Deliberately no Unwrap(): everything must go
// through the mutex.
func (w *mutexWriter) Flush() {
	w.mut.Lock()
	defer w.mut.Unlock()
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// withKeepalive sends ":\n" every 15s in a background goroutine and
// mutex-protects all writes. The caller must cancel ctx to stop it.
func withKeepalive(ctx context.Context, w http.ResponseWriter) http.ResponseWriter {
	mut := &sync.Mutex{}
	go keepalive(ctx, w, mut)
	return &mutexWriter{ResponseWriter: w, mut: mut}
}

// StreamForwarder is a grpc-gateway v2 compatible stream forwarder that supports
// SSE formatting and field filtering via the ?fields query parameter.
var StreamForwarder = func(
	ctx context.Context,
	mux *runtime.ServeMux,
	marshaler runtime.Marshaler,
	w http.ResponseWriter,
	req *http.Request,
	recv func() (proto.Message, error),
	opts ...func(context.Context, http.ResponseWriter, proto.Message) error,
) {
	isSSE := strings.Contains(req.Header.Get("Accept"), "text/event-stream")
	processCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	if isSSE {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w = withKeepalive(processCtx, w)
	}
	m := newStreamMarshaler(req, isSSE)
	runtime.ForwardResponseStream(ctx, mux, m, w, req, recv, opts...)
}
