package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func newTestMarshaler(t *testing.T, query string, isSSE bool) *messageMarshaler {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/workflows"+query, nil)
	return newStreamMarshaler(req, isSSE)
}

// shortKeepalive shrinks the keepalive interval for the duration of a test.
func shortKeepalive(t *testing.T, d time.Duration) {
	t.Helper()
	old := keepaliveInterval
	keepaliveInterval = d
	t.Cleanup(func() { keepaliveInterval = old })
}

func TestMessageMarshaler_ContentType(t *testing.T) {
	m := &messageMarshaler{isSSE: false}
	assert.Equal(t, "application/json", m.ContentType(nil))

	m = &messageMarshaler{isSSE: true}
	assert.Equal(t, "text/event-stream", m.ContentType(nil))
}

func TestMessageMarshaler_OnlyMarshalSupported(t *testing.T) {
	m := newTestMarshaler(t, "", false)
	require.Error(t, m.Unmarshal([]byte("{}"), nil))
	require.Error(t, m.NewDecoder(nil).Decode(nil))
	require.Error(t, m.NewEncoder(nil).Encode(nil))
}

// The ?fields filtering semantics themselves are covered by util/fields' own
// tests; these cases pin that the query parameter is wired into the marshaler
// and that paths are relative to grpc-gateway's {"result": ...} envelope.
func TestMessageMarshaler_Marshal(t *testing.T) {
	input := map[string]any{"result": map[string]any{
		"name":   "test",
		"status": "running",
	}}

	t.Run("no fields", func(t *testing.T) {
		m := newTestMarshaler(t, "", false)
		data, err := m.Marshal(input)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"test"`)
		assert.Contains(t, string(data), `"status":"running"`)
	})

	t.Run("include fields", func(t *testing.T) {
		m := newTestMarshaler(t, "?fields=result.name", false)
		data, err := m.Marshal(input)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"test"`)
		assert.NotContains(t, string(data), `"status"`)
	})
}

func TestMessageMarshaler_Marshal_SSE(t *testing.T) {
	m := newTestMarshaler(t, "", true)
	input := map[string]any{"result": map[string]any{"name": "test"}}

	data, err := m.Marshal(input)
	require.NoError(t, err)

	s := string(data)
	assert.Contains(t, s, "data: ")
	assert.Contains(t, s, `"name":"test"`)
	assert.Equal(t, "\n\n", s[len(s)-2:], "SSE data should end with double newline")
}

// errorWriter is an http.ResponseWriter that always returns an error on Write.
type errorWriter struct {
	header http.Header
}

func newErrorWriter() *errorWriter {
	return &errorWriter{header: make(http.Header)}
}

func (e *errorWriter) Header() http.Header       { return e.header }
func (e *errorWriter) WriteHeader(int)           {}
func (e *errorWriter) Write([]byte) (int, error) { return 0, errors.New("connection closed") }

func TestWriteKeepalive_Success(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	rec := httptest.NewRecorder()
	mut := &sync.Mutex{}

	ok := writeKeepalive(ctx, rec, mut)

	assert.True(t, ok)
	assert.Equal(t, ":\n", rec.Body.String())
	assert.True(t, rec.Flushed, "keepalive should flush the writer")
}

func TestWriteKeepalive_Failure(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	w := newErrorWriter()
	mut := &sync.Mutex{}

	ok := writeKeepalive(ctx, w, mut)

	assert.False(t, ok)
}

// A request context without a logger must not panic: production HTTP request
// contexts may not carry one, and the keepalive runs in a background goroutine
// where a panic would kill the process.
func TestWriteKeepalive_NoLoggerInContext(t *testing.T) {
	w := newErrorWriter()
	mut := &sync.Mutex{}

	assert.NotPanics(t, func() {
		ok := writeKeepalive(context.Background(), w, mut) //nolint:testingcontext // deliberately logger-free
		assert.False(t, ok)
	})
}

func TestKeepalive_StopsOnWriteError(t *testing.T) {
	shortKeepalive(t, time.Millisecond)
	w := newErrorWriter()
	mut := &sync.Mutex{}

	done := make(chan struct{})
	go func() {
		// Deliberately logger-free context: the failure path must not panic.
		keepalive(context.Background(), w, mut) //nolint:testingcontext
		close(done)
	}()

	select {
	case <-done:
		// keepalive returned after the first failed write
	case <-time.After(2 * time.Second):
		t.Fatal("keepalive goroutine did not stop on write error")
	}
}

func TestKeepalive_StopsOnContextCancel(t *testing.T) {
	rec := httptest.NewRecorder()
	mut := &sync.Mutex{}
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context())) //nolint:testingcontext

	done := make(chan struct{})
	go func() {
		keepalive(ctx, rec, mut)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// Goroutine stopped as expected
	case <-time.After(2 * time.Second):
		t.Fatal("keepalive goroutine did not stop on context cancellation")
	}
}

// syncRecorder is a ResponseWriter safe to read while the keepalive goroutine
// (which holds the raw writer) is writing to it.
type syncRecorder struct {
	mu     sync.Mutex
	buf    strings.Builder
	header http.Header
}

func newSyncRecorder() *syncRecorder { return &syncRecorder{header: make(http.Header)} }

func (s *syncRecorder) Header() http.Header { return s.header }
func (s *syncRecorder) WriteHeader(int)     {}
func (s *syncRecorder) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *syncRecorder) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

func TestWithKeepalive_EmitsKeepalives(t *testing.T) {
	shortKeepalive(t, time.Millisecond)
	rec := newSyncRecorder()
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context())) //nolint:testingcontext
	defer cancel()

	_ = withKeepalive(ctx, rec)

	assert.Eventually(t, func() bool {
		return strings.Contains(rec.String(), ":\n")
	}, 2*time.Second, 5*time.Millisecond, "keepalive frames should reach the writer")
}

// flushRecorder counts Flush calls so tests can assert the wrapper forwards them.
type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (f *flushRecorder) Flush() { f.flushes++ }

func TestMutexWriter_SupportsResponseController(t *testing.T) {
	// grpc-gateway's ForwardResponseStream flushes after every message via
	// http.NewResponseController. The keepalive wrapper must not hide the
	// underlying writer's Flusher, or every SSE stream aborts with
	// "unexpected type of web server".
	rec := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	w := &mutexWriter{ResponseWriter: rec, mut: &sync.Mutex{}}

	rc := http.NewResponseController(w)
	require.NoError(t, rc.Flush())
	assert.Equal(t, 1, rec.flushes)

	_, err := w.Write([]byte("data"))
	require.NoError(t, err)
	assert.Equal(t, "data", rec.Body.String())
}

func TestWithKeepalive_ConcurrentWrites(t *testing.T) {
	// The mutexWriter must serialize writes; with the interval shortened the
	// keepalive goroutine contends with the handler writes below.
	shortKeepalive(t, time.Millisecond)
	rec := newSyncRecorder()
	ctx, cancel := context.WithCancel(logging.TestContext(t.Context())) //nolint:testingcontext
	defer cancel()
	w := withKeepalive(ctx, rec)

	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			_, err := w.Write([]byte("x"))
			assert.NoError(t, err)
		})
	}
	wg.Wait()
	assert.Equal(t, 10, strings.Count(rec.String(), "x"))
}
