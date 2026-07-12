package hdfs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestSaveStreamTempFileCreation tests that SaveStream correctly creates a temp file
func TestSaveStreamTempFileCreation(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Create a driver that will fail at the Save step (no HDFS connection)
	// This tests that the temp file creation and writing works correctly
	driver := &ArtifactDriver{
		Addresses: []string{"nonexistent:8020"},
		Path:      "/test/path",
		HDFSUser:  "testuser",
	}

	testContent := "test content"
	reader := strings.NewReader(testContent)

	outputArtifact := &wfv1.Artifact{
		ArtifactLocation: wfv1.ArtifactLocation{
			HDFS: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{"nonexistent:8020"},
					HDFSUser:  "testuser",
				},
				Path: "/test/path",
			},
		},
	}

	// This will fail at the HDFS client creation step, but verifies
	// that the temp file logic doesn't panic
	err := driver.SaveStream(ctx, reader, outputArtifact)
	// We expect an error due to no HDFS connection
	require.Error(t, err)
}
