package plugin

import (
	"bytes"
	"context"
	"errors"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

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
	saveCalled bool
	savePath   string
}

func (m *mockArtifactServer) GetCapabilities(_ context.Context, _ *artifact.GetCapabilitiesRequest) (*artifact.GetCapabilitiesResponse, error) {
	return &artifact.GetCapabilitiesResponse{SupportsSaveStream: m.supportsSaveStream}, nil
}

func (m *mockArtifactServer) SaveStream(stream artifact.ArtifactService_SaveStreamServer) error {
	first := true
	for {
		req, err := stream.Recv()
		if errors.Is(err, context.Canceled) {
			return err
		}
		if err != nil {
			if err.Error() == "EOF" {
				return stream.SendAndClose(&artifact.SaveArtifactResponse{Success: true})
			}
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
	return &artifact.SaveArtifactResponse{Success: true}, nil
}

// startMockPluginServer starts a gRPC server serving mockArtifactServer over a unix
// socket and returns a connected Driver, cleaning both up on test completion.
func startMockPluginServer(t *testing.T, mock *mockArtifactServer) *Driver {
	t.Helper()
	ctx := logging.TestContext(t.Context())

	socketPath := filepath.Join(t.TempDir(), "plugin.sock")
	listener, err := net.Listen("unix", socketPath)
	require.NoError(t, err)

	server := grpc.NewServer()
	artifact.RegisterArtifactServiceServer(server, mock)
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

		content := bytes.Repeat([]byte("abcdefghij"), 20000) // larger than one chunk
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
	})

	t.Run("mid-stream failure is returned as an error, not a fallback", func(t *testing.T) {
		mock := &mockArtifactServer{
			supportsSaveStream:         true,
			saveStreamErrAfterMetadata: errors.New("simulated mid-stream failure"),
		}
		driver := startMockPluginServer(t, mock)
		ctx := logging.TestContext(t.Context())

		err := driver.SaveStream(ctx, bytes.NewReader([]byte("content that will fail partway")), outputArtifact)
		require.Error(t, err)
		assert.False(t, mock.saveCalled, "must not silently fall back to Save after streaming has started")
	})
}
