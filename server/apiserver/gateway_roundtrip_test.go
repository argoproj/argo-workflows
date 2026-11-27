package apiserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	workflowpkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/workflow"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// This file holds an HTTP round-trip test over the real generated grpc-gateway
// code: HTTP request -> gateway mux -> gRPC (bufconn) -> service -> response.
// It pins the seams that unit tests cannot see:
//   - unary responses of gogo-generated types must serialize as real JSON, not
//     {} (the gateway.MessageV2Of bridge injected by the Makefile's protoc rule),
//   - SSE streams must actually stream: the keepalive writer must keep Flush
//     reachable for grpc-gateway's http.ResponseController, and each message
//     arrives in the {"result": ...} envelope the UI depends on,
//   - errors surface in the google.rpc.Status shape docs/upgrading.md promises,
//     for both unary responses and in-stream error chunks.

// fakeWorkflowServer embeds the Unimplemented stub for brevity; production
// servers implement the interface explicitly (see the protoc rule's
// require_unimplemented_servers comment in the Makefile).
type fakeWorkflowServer struct {
	workflowpkg.UnimplementedWorkflowServiceServer
}

func (s *fakeWorkflowServer) GetWorkflow(_ context.Context, req *workflowpkg.WorkflowGetRequest) (*wfv1.Workflow, error) {
	if req.Name == "missing" {
		return nil, status.Error(codes.NotFound, "workflow missing not found")
	}
	return &wfv1.Workflow{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: req.Namespace},
		Spec:       wfv1.WorkflowSpec{Entrypoint: "main"},
	}, nil
}

func (s *fakeWorkflowServer) WatchEvents(req *workflowpkg.WatchEventsRequest, ws grpc.ServerStreamingServer[workflowpkg.EventWatchEvent]) error {
	for i, eventType := range []string{"ADDED", "DELETED"} {
		if err := ws.Send(&workflowpkg.EventWatchEvent{
			Type: eventType,
			Object: &corev1.Event{
				ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("event-%d", i), Namespace: req.Namespace},
				Message:    fmt.Sprintf("message-%d", i),
			},
		}); err != nil {
			return err
		}
	}
	if req.Namespace == "stream-error" {
		return status.Error(codes.PermissionDenied, "watch denied")
	}
	return nil
}

func (s *fakeWorkflowServer) WorkflowLogs(req *workflowpkg.WorkflowLogRequest, ws grpc.ServerStreamingServer[workflowpkg.LogEntry]) error {
	// Echo the query-populated options back so the test can verify population.
	return ws.Send(&workflowpkg.LogEntry{
		Content: fmt.Sprintf("container=%s follow=%v", req.LogOptions.Container, req.LogOptions.Follow),
	})
}

// newGatewayServer starts a real gRPC server on a bufconn listener, registers
// the generated gateway handlers against the production mux configuration
// (newGatewayMux), and serves it wrapped exactly as newHTTPServer does
// (withRequestLogger).
func newGatewayServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := logging.TestContext(t.Context())

	lis := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	workflowpkg.RegisterWorkflowServiceServer(grpcServer, &fakeWorkflowServer{})
	go func() { _ = grpcServer.Serve(lis) }()
	t.Cleanup(grpcServer.Stop)

	gwmux := newGatewayMux()
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
	}
	require.NoError(t, workflowpkg.RegisterWorkflowServiceHandlerFromEndpoint(ctx, gwmux, "passthrough:///bufconn", dialOpts))

	httpServer := httptest.NewServer(withRequestLogger(logging.RequireLoggerFromContext(ctx), gwmux))
	t.Cleanup(httpServer.Close)
	return httpServer
}

// get performs a GET against the test server, optionally as an SSE request,
// and returns the status code, response headers, and body. The response is
// fully read and closed in here so callers cannot leak it.
func get(t *testing.T, server *httptest.Server, path string, sse bool) (int, http.Header, string) {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+path, nil)
	require.NoError(t, err)
	if sse {
		req.Header.Set("Accept", "text/event-stream")
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, resp.Header, string(body)
}

func TestGatewayRoundtrip_UnaryGogoMessageJSON(t *testing.T) {
	server := newGatewayServer(t)

	status, _, body := get(t, server, "/api/v1/workflows/argo/my-wf", false)

	require.Equal(t, http.StatusOK, status, "body: %s", body)
	// A gogo-generated response must serialize as real JSON. A bare protoadapt
	// wrapper has no exported fields and would produce exactly "{}".
	assert.NotEqual(t, "{}", strings.TrimSpace(body))
	assert.Contains(t, body, `"name":"my-wf"`)
	assert.Contains(t, body, `"namespace":"argo"`)
	assert.Contains(t, body, `"entrypoint":"main"`)
}

func TestGatewayRoundtrip_UnaryErrorStatusShape(t *testing.T) {
	server := newGatewayServer(t)

	status, _, body := get(t, server, "/api/v1/workflows/argo/missing", false)

	// docs/upgrading.md: HTTP error bodies use the google.rpc.Status shape.
	require.Equal(t, http.StatusNotFound, status, "body: %s", body)
	assert.Contains(t, body, fmt.Sprintf(`"code":%d`, codes.NotFound))
	assert.Contains(t, body, `"message":"workflow missing not found"`)
	assert.NotContains(t, body, `"error":`, "grpc-gateway v1's error field must be gone")
}

func TestGatewayRoundtrip_SSEStream(t *testing.T) {
	server := newGatewayServer(t)

	status, header, s := get(t, server, "/api/v1/stream/events/argo", true)

	require.Equal(t, http.StatusOK, status, "body: %s", s)
	assert.Equal(t, "text/event-stream", header.Get("Content-Type"))
	// The keepalive writer must not hide the underlying Flusher, or the gateway
	// aborts the stream with this error after the first message.
	assert.NotContains(t, s, "unexpected type of web server")
	// Both messages arrive as SSE frames in the {"result": ...} envelope, with
	// the EventWatchEvent {type, object} shape the UI unwraps — including the
	// event type passed through from the server, not a hardcoded ADDED.
	assert.Equal(t, 2, strings.Count(s, "data: "), "expected two SSE frames, body: %s", s)
	assert.Contains(t, s, `"result":{"type":"ADDED","object":`)
	assert.Contains(t, s, `"result":{"type":"DELETED","object":`)
	assert.Contains(t, s, `"message":"message-0"`)
	assert.Contains(t, s, `"message":"message-1"`)
}

func TestGatewayRoundtrip_StreamErrorStatusShape(t *testing.T) {
	server := newGatewayServer(t)

	status, _, s := get(t, server, "/api/v1/stream/events/stream-error", true)

	// A mid-stream failure arrives as a trailing {"error": ...} chunk in the
	// google.rpc.Status shape (headers were already sent, so still HTTP 200).
	require.Equal(t, http.StatusOK, status, "body: %s", s)
	assert.Contains(t, s, `"message":"message-1"`, "messages before the error must still arrive")
	assert.Contains(t, s, `"error":`)
	assert.Contains(t, s, fmt.Sprintf(`"code":%d`, codes.PermissionDenied))
	assert.Contains(t, s, `"message":"watch denied"`)
	assert.NotContains(t, s, `"grpc_code"`, "grpc-gateway v1's stream error fields must be gone")
}

func TestGatewayRoundtrip_StreamFieldFilter(t *testing.T) {
	server := newGatewayServer(t)

	status, _, s := get(t, server, "/api/v1/stream/events/argo?fields=result.object.message", true)

	require.Equal(t, http.StatusOK, status, "body: %s", s)
	assert.Equal(t, 2, strings.Count(s, "data: "), "filtering must not break streaming, body: %s", s)
	assert.Contains(t, s, `"message":"message-0"`)
	assert.Contains(t, s, `"message":"message-1"`)
	assert.NotContains(t, s, `"type":"ADDED"`, "?fields should have filtered out result.type")
	assert.NotContains(t, s, `"metadata"`, "?fields should have filtered out result.object.metadata")
}

func TestGatewayRoundtrip_NonSSEStream(t *testing.T) {
	server := newGatewayServer(t)

	status, _, s := get(t, server, "/api/v1/stream/events/argo", false)

	require.Equal(t, http.StatusOK, status, "body: %s", s)
	assert.NotContains(t, s, "data: ", "non-SSE stream should be newline-delimited JSON, not SSE frames")
	assert.Contains(t, s, `"result":{"type":"ADDED","object":`)
	assert.Equal(t, 2, strings.Count(s, `"message":"message-`))
}

// The gateway populates ?logOptions.*= query parameters into the embedded
// corev1.PodLogOptions via protoreflect, which derives descriptors for the
// Kubernetes type from its struct tags. Kubernetes mis-tags the `stream` field
// (varint for a *string); without the hack/vendor-patches.sh tag fix, every
// log request with logOptions parameters panics the handler.
func TestGatewayRoundtrip_LogQueryParamsPopulateK8sOptions(t *testing.T) {
	server := newGatewayServer(t)

	status, _, s := get(t, server, "/api/v1/workflows/argo/my-wf/log?logOptions.container=main&logOptions.follow=true&logOptions.stream=All", false)

	require.Equal(t, http.StatusOK, status, "body: %s", s)
	assert.Contains(t, s, "container=main follow=true")
}
