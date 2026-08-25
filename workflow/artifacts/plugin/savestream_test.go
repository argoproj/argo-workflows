package plugin

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient/artifact"
	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// mockArtifactServer is a minimal ArtifactServiceServer used to exercise the plugin
// Driver's SaveStream implementation against a real gRPC connection over a unix socket.
type mockArtifactServer struct {
	artifact.UnimplementedArtifactServiceServer

	supportsSaveStream bool

	// saveStreamErrAfterMetadata, if set, is returned by SaveStream right after the
	// first (metadata) frame is received, simulating a mid-stream failure.
	saveStreamErrAfterMetadata error

	// received captures the chunks assembled from a successful SaveStream call.
	received bytes.Buffer

	// saveCalled records whether the unary Save RPC was invoked (fallback path).
	saveCalled   bool
	savePath     string
	savedContent []byte

	// serverCanceled, if non-nil, is closed when the server's Recv() observes
	// context.Canceled, confirming the client released the stream instead of
	// leaving the server blocked in Recv().
	serverCanceled chan struct{}
}

func (m *mockArtifactServer) GetCapabilities(_ context.Context, _ *artifact.GetCapabilitiesRequest) (*artifact.GetCapabilitiesResponse, error) {
	return &artifact.GetCapabilitiesResponse{SupportsSaveStream: m.supportsSaveStream}, nil
}

func (m *mockArtifactServer) SaveStream(stream artifact.ArtifactService_SaveStreamServer) error {
	first := true
	for {
		req, err := stream.Recv()
		// A client-side ctx cancel surfaces here as a gRPC status error (codes.Canceled),
		// not as context.Canceled itself, since it crossed the wire.
		if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
			if m.serverCanceled != nil {
				close(m.serverCanceled)
			}
			return err
		}
		if errors.Is(err, io.EOF) {
			return stream.SendAndClose(&artifact.SaveArtifactResponse{Success: true})
		}
		if err != nil {
			return err
		}
		if first {
			first = false
			if m.saveStreamErrAfterMetadata != nil {
				return m.saveStreamErrAfterMetadata
			}
			continue
		}
		m.received.Write(req.GetChunk())
	}
}

func (m *mockArtifactServer) Save(_ context.Context, req *artifact.SaveArtifactRequest) (*artifact.SaveArtifactResponse, error) {
	m.saveCalled = true
	m.savePath = req.GetPath()
	// Server and driver share a filesystem in-test, so read back the buffered file
	// to prove the fallback wrote the reader's content, not an empty/truncated file.
	if data, err := os.ReadFile(req.GetPath()); err == nil {
		m.savedContent = data
	}
	return &artifact.SaveArtifactResponse{Success: true}, nil
}

// legacyMockServer implements only Save, leaving GetCapabilities and SaveStream
// unimplemented so the gRPC framework returns codes.Unimplemented for them —
// exactly how a plugin built before streaming existed behaves.
type legacyMockServer struct {
	artifact.UnimplementedArtifactServiceServer
	saveCalled   bool
	savedContent []byte
}

func (m *legacyMockServer) Save(_ context.Context, req *artifact.SaveArtifactRequest) (*artifact.SaveArtifactResponse, error) {
	m.saveCalled = true
	if data, err := os.ReadFile(req.GetPath()); err == nil {
		m.savedContent = data
	}
	return &artifact.SaveArtifactResponse{Success: true}, nil
}

// startMockPluginServer starts a gRPC server serving srv over a unix socket and
// returns a connected Driver, cleaning both up on test completion.
func startMockPluginServer(t *testing.T, srv artifact.ArtifactServiceServer) *Driver {
	t.Helper()
	if runtime.GOOS == "windows" {
		// Artifact plugins communicate over a unix socket, which Windows does not
		// support (and plugins are unsupported on Windows).
		t.Skip("plugin artifact driver is not supported on Windows")
	}
	ctx := logging.TestContext(t.Context())

	socketPath := filepath.Join(t.TempDir(), "plugin.sock")
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "unix", socketPath)
	require.NoError(t, err)

	server := grpc.NewServer()
	artifact.RegisterArtifactServiceServer(server, srv)
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(server.Stop)

	driver, err := NewDriver(ctx, "test-plugin", socketPath, 5*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = driver.Close() })

	return driver
}

func TestDriverSaveStream(t *testing.T) {
	outputArtifact := &wfv1.Artifact{
		Name: "test-artifact",
		ArtifactLocation: wfv1.ArtifactLocation{
			Plugin: &wfv1.PluginArtifact{Name: "test-plugin", Key: "test-key"},
		},
	}

	t.Run("streams chunks when the plugin supports SaveStream", func(t *testing.T) {
		mock := &mockArtifactServer{supportsSaveStream: true}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		content := bytes.Repeat([]byte("abcdefghij"), 600000) // 6MB: spans multiple chunks
		err := driver.SaveStream(ctx, bytes.NewReader(content), outputArtifact)
		require.NoError(t, err)
		assert.Equal(t, content, mock.received.Bytes())
		assert.False(t, mock.saveCalled, "unary Save must not be called when streaming succeeds")
	})

	t.Run("falls back to Save when capability is false", func(t *testing.T) {
		mock := &mockArtifactServer{supportsSaveStream: false}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		err := driver.SaveStream(ctx, bytes.NewReader([]byte("fallback content")), outputArtifact)
		require.NoError(t, err)
		assert.True(t, mock.saveCalled, "unary Save must be called as the fallback")
		assert.NotEmpty(t, mock.savePath)
		assert.Equal(t, "fallback content", string(mock.savedContent),
			"the buffered file handed to Save must hold the reader's content")
	})

	t.Run("falls back to Save when the plugin does not implement GetCapabilities", func(t *testing.T) {
		// A pre-streaming plugin has no GetCapabilities method, so the gRPC framework
		// returns codes.Unimplemented, which must map to the buffered Save fallback.
		mock := &legacyMockServer{}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		err := driver.SaveStream(ctx, bytes.NewReader([]byte("legacy content")), outputArtifact)
		require.NoError(t, err)
		assert.True(t, mock.saveCalled, "unary Save must be called when GetCapabilities is Unimplemented")
		assert.Equal(t, "legacy content", string(mock.savedContent))
	})

	t.Run("mid-stream failure is returned as an error, not a fallback", func(t *testing.T) {
		mock := &mockArtifactServer{
			supportsSaveStream:         true,
			saveStreamErrAfterMetadata: errors.New("simulated mid-stream failure"),
		}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		// Multi-chunk payload: once the server aborts, a subsequent Send returns
		// io.EOF, so this exercises the CloseAndRecv recovery of the real error.
		content := bytes.Repeat([]byte("content that will fail partway"), 300000) // ~9MB
		err := driver.SaveStream(ctx, bytes.NewReader(content), outputArtifact)
		require.ErrorContains(t, err, "simulated mid-stream failure",
			"the plugin's actual error must surface, not a bare EOF")
		assert.False(t, mock.saveCalled, "must not silently fall back to Save after streaming has started")
	})

	t.Run("reader error cancels the stream instead of leaking it", func(t *testing.T) {
		mock := &mockArtifactServer{supportsSaveStream: true, serverCanceled: make(chan struct{})}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		readerErr := errors.New("simulated reader failure")
		err := driver.SaveStream(ctx, &erroringReader{err: readerErr}, outputArtifact)
		require.ErrorIs(t, err, readerErr)

		select {
		case <-mock.serverCanceled:
		case <-time.After(5 * time.Second):
			t.Fatal("server never observed stream cancellation; SaveStream leaked the stream on a reader error")
		}
	})
}

// erroringReader is an io.Reader whose Read always fails, simulating a reader that
// breaks partway through a SaveStream call.
type erroringReader struct {
	err error
}

func (r *erroringReader) Read([]byte) (int, error) {
	return 0, r.err
}
