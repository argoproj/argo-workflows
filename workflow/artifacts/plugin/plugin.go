package plugin

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient/artifact"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// Driver implements the ArtifactDriver interface by making gRPC calls to a plugin service
type Driver struct {
	pluginName wfv1.ArtifactPluginName
	conn       *grpc.ClientConn
	client     artifact.ArtifactServiceClient
}

// NewDriver creates a new plugin artifact driver
func NewDriver(ctx context.Context, pluginName wfv1.ArtifactPluginName, socketPath string, connectionTimeout time.Duration) (*Driver, error) {
	// Check for the unix socket, retrying for up to two minutes if it doesn't exist immediately
	logger := logging.RequireLoggerFromContext(ctx)

	// Try for up to 120 seconds, checking once per second
	const maxRetries = 120
	var info os.FileInfo
	var statErr error
	var socketExists bool

	for retry := range maxRetries {
		info, statErr = os.Stat(socketPath)
		if statErr == nil {
			socketExists = true
			break
		}

		if !os.IsNotExist(statErr) {
			// If error is not due to missing file, fail immediately
			return nil, fmt.Errorf("plugin %s cannot stat unix socket at %q: %w", pluginName, socketPath, statErr)
		}

		// Socket doesn't exist yet, log at debug level and retry
		logger.WithFields(logging.Fields{
			"pluginName": pluginName,
			"socketPath": socketPath,
			"retry":      retry,
			"maxRetries": maxRetries,
		}).Debug(ctx, "plugin socket not found, retrying in 1s")

		// Use context-aware sleep
		select {
		case <-time.After(time.Second):
			// Continue to next iteration
		case <-ctx.Done():
			return nil, fmt.Errorf("plugin %s context cancelled while waiting for socket at %q: %w", pluginName, socketPath, ctx.Err())
		}
	}

	// If socket still doesn't exist after all retries, fail with error
	if !socketExists {
		return nil, fmt.Errorf("plugin %s expected unix socket at %q but it does not exist after waiting for %d seconds", pluginName, socketPath, maxRetries)
	}

	if (info.Mode() & os.ModeSocket) == 0 {
		logger.WithFields(logging.Fields{
			"pluginName": pluginName,
			"socketPath": socketPath,
			"mode":       info.Mode(),
		}).Warn(ctx, "plugin socket file exists but is not a unix socket")
	}
	logger.WithFields(logging.Fields{
		"pluginName": pluginName,
		"socketPath": socketPath,
		"mode":       info.Mode(),
	}).Info(ctx, "plugin socket file exists and is a unix socket")

	conn, err := grpc.NewClient(
		"unix://"+socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			// Strip unix:// prefix if present
			if len(addr) > 7 && addr[:7] == "unix://" {
				addr = addr[7:]
			}
			dialer := &net.Dialer{Timeout: connectionTimeout}
			return dialer.DialContext(ctx, "unix", addr)
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial plugin %s at %q: %w", pluginName, socketPath, err)
	}

	driver := &Driver{
		pluginName: pluginName,
		conn:       conn,
		client:     artifact.NewArtifactServiceClient(conn),
	}

	// Verify the connection by checking the connection state
	ctx, cancel := context.WithTimeout(ctx, connectionTimeout)
	defer cancel()

	conn.Connect()

	// Wait for the connection to reach Ready state within the timeout
	for state := conn.GetState(); state != connectivity.Ready; state = conn.GetState() {
		if state == connectivity.Shutdown {
			_ = conn.Close()
			return nil, fmt.Errorf("plugin %s connection shutdown (socket=%q)", pluginName, socketPath)
		}
		if !conn.WaitForStateChange(ctx, state) {
			// Timeout or context cancelled
			currentState := conn.GetState()
			_ = conn.Close()
			return nil, fmt.Errorf("timeout waiting for plugin %s connection to become ready, last state: %s (socket=%q)", pluginName, currentState, socketPath)
		}
	}

	logger.Info(ctx, fmt.Sprintf("plugin %s: connected successfully to %q", pluginName, socketPath))
	return driver, nil
}

// Close closes the gRPC connection
func (d *Driver) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// Load implements ArtifactDriver.Load by calling the plugin service
func (d *Driver) Load(ctx context.Context, inputArtifact *wfv1.Artifact, path string) error {
	grpcArtifact := convertToGRPC(inputArtifact)
	resp, err := d.client.Load(ctx, &artifact.LoadArtifactRequest{
		InputArtifact: grpcArtifact,
		Path:          path,
	})
	if err != nil {
		return fmt.Errorf("plugin %s load failed: %w", d.pluginName, err)
	}
	if !resp.Success {
		return fmt.Errorf("plugin %s load failed: %s", d.pluginName, resp.Error)
	}
	return nil
}

// OpenStream implements ArtifactDriver.OpenStream by calling the plugin service
func (d *Driver) OpenStream(ctx context.Context, a *wfv1.Artifact) (io.ReadCloser, error) {
	grpcArtifact := convertToGRPC(a)
	stream, err := d.client.OpenStream(ctx, &artifact.OpenStreamRequest{
		Artifact: grpcArtifact,
	})
	if err != nil {
		return nil, fmt.Errorf("plugin %s open stream failed: %w", d.pluginName, err)
	}

	reader, writer := io.Pipe()

	go func() {
		defer writer.Close()
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				writer.CloseWithError(fmt.Errorf("plugin %s stream receive failed: %w", d.pluginName, err))
				return
			}
			if resp.Error != "" {
				writer.CloseWithError(fmt.Errorf("plugin %s stream error: %s", d.pluginName, resp.Error))
				return
			}
			if resp.IsEnd {
				break
			}
			if len(resp.Data) > 0 {
				if _, writeErr := writer.Write(resp.Data); writeErr != nil {
					writer.CloseWithError(fmt.Errorf("plugin %s stream write failed: %w", d.pluginName, writeErr))
					return
				}
			}
		}
	}()

	return reader, nil
}

// Save implements ArtifactDriver.Save by calling the plugin service
func (d *Driver) Save(ctx context.Context, path string, outputArtifact *wfv1.Artifact) error {
	grpcArtifact := convertToGRPC(outputArtifact)
	resp, err := d.client.Save(ctx, &artifact.SaveArtifactRequest{
		Path:           path,
		OutputArtifact: grpcArtifact,
	})
	if err != nil {
		return fmt.Errorf("plugin %s save failed: %w", d.pluginName, err)
	}
	if !resp.Success {
		return fmt.Errorf("plugin %s save failed: %s", d.pluginName, resp.Error)
	}
	return nil
}

// SaveStream implements ArtifactDriver.SaveStream by using a temporary file
// Note: gRPC streaming upload is complex, so we use a temp file fallback
func (d *Driver) SaveStream(ctx context.Context, reader io.Reader, outputArtifact *wfv1.Artifact) error {
	// Write to temp file first
	tmpFile, err := os.CreateTemp("", "plugin-upload-*")
	if err != nil {
		return fmt.Errorf("plugin %s failed to create temp file: %w", d.pluginName, err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err = io.Copy(tmpFile, reader); err != nil {
		tmpFile.Close()
		return fmt.Errorf("plugin %s failed to write to temp file: %w", d.pluginName, err)
	}
	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("plugin %s failed to close temp file: %w", d.pluginName, err)
	}
	return d.Save(ctx, tmpFile.Name(), outputArtifact)
}

// Delete implements ArtifactDriver.Delete by calling the plugin service
func (d *Driver) Delete(ctx context.Context, artifactRef *wfv1.Artifact) error {
	grpcArtifact := convertToGRPC(artifactRef)
	resp, err := d.client.Delete(ctx, &artifact.DeleteArtifactRequest{
		Artifact: grpcArtifact,
	})
	if err != nil {
		return fmt.Errorf("plugin %s delete failed: %w", d.pluginName, err)
	}
	if !resp.Success {
		return fmt.Errorf("plugin %s delete failed: %s", d.pluginName, resp.Error)
	}
	return nil
}

// ListObjects implements ArtifactDriver.ListObjects by calling the plugin service
func (d *Driver) ListObjects(ctx context.Context, artifactRef *wfv1.Artifact) ([]string, error) {
	grpcArtifact := convertToGRPC(artifactRef)
	resp, err := d.client.ListObjects(ctx, &artifact.ListObjectsRequest{
		Artifact: grpcArtifact,
	})
	if err != nil {
		return nil, fmt.Errorf("plugin %s list objects failed: %w", d.pluginName, err)
	}
	if resp.Error != "" {
		return nil, fmt.Errorf("plugin %s list objects failed: %s", d.pluginName, resp.Error)
	}
	return resp.Objects, nil
}

// IsDirectory implements ArtifactDriver.IsDirectory by calling the plugin service
func (d *Driver) IsDirectory(ctx context.Context, artifactRef *wfv1.Artifact) (bool, error) {
	grpcArtifact := convertToGRPC(artifactRef)
	resp, err := d.client.IsDirectory(ctx, &artifact.IsDirectoryRequest{
		Artifact: grpcArtifact,
	})
	if err != nil {
		return false, fmt.Errorf("plugin %s is directory check failed: %w", d.pluginName, err)
	}
	if resp.Error != "" {
		return false, fmt.Errorf("plugin %s is directory check failed: %s", d.pluginName, resp.Error)
	}
	return resp.IsDirectory, nil
}

// convertToGRPC converts v1alpha1.Artifact to gRPC Artifact
func convertToGRPC(a *wfv1.Artifact) *artifact.Artifact {
	if a == nil {
		return nil
	}

	grpcArtifact := &artifact.Artifact{
		Name:           a.Name,
		Path:           a.Path,
		From:           a.From,
		Optional:       a.Optional,
		SubPath:        a.SubPath,
		RecurseMode:    a.RecurseMode,
		FromExpression: a.FromExpression,
		Deleted:        a.Deleted,
	}
	if a.Mode != nil {
		grpcArtifact.Mode = *a.Mode
	}

	if a.Plugin != nil {
		grpcArtifact.Plugin = &artifact.PluginArtifact{
			Name:                     string(a.Plugin.Name),
			Configuration:            a.Plugin.Configuration,
			ConnectionTimeoutSeconds: a.Plugin.ConnectionTimeoutSeconds,
			Key:                      a.Plugin.Key,
		}
	}
	return grpcArtifact
}
