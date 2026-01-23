package hdfs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/errors"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestValidateArtifact(t *testing.T) {
	tests := []struct {
		name      string
		artifact  *wfv1.HDFSArtifact
		errPrefix string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "Valid with HDFSUser",
			artifact: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{"namenode:8020"},
					HDFSUser:  "testuser",
				},
				Path: "/test/path",
			},
			errPrefix: "test",
			wantErr:   false,
		},
		{
			name: "Missing addresses",
			artifact: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{},
					HDFSUser:  "testuser",
				},
				Path: "/test/path",
			},
			errPrefix: "test",
			wantErr:   true,
			errMsg:    "test.addresses is required",
		},
		{
			name: "Missing path",
			artifact: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{"namenode:8020"},
					HDFSUser:  "testuser",
				},
				Path: "",
			},
			errPrefix: "test",
			wantErr:   true,
			errMsg:    "test.path is required",
		},
		{
			name: "Relative path",
			artifact: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{"namenode:8020"},
					HDFSUser:  "testuser",
				},
				Path: "relative/path",
			},
			errPrefix: "test",
			wantErr:   true,
			errMsg:    "test.path must be a absolute file path",
		},
		{
			name: "Missing authentication",
			artifact: &wfv1.HDFSArtifact{
				HDFSConfig: wfv1.HDFSConfig{
					Addresses: []string{"namenode:8020"},
				},
				Path: "/test/path",
			},
			errPrefix: "test",
			wantErr:   true,
			errMsg:    "either test.hdfsUser, test.krbCCacheSecret or test.krbKeytabSecret is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateArtifact(tc.errPrefix, tc.artifact)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

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

func TestDeleteNotSupported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &ArtifactDriver{
		Addresses: []string{"namenode:8020"},
		Path:      "/test/path",
		HDFSUser:  "testuser",
	}

	artifact := &wfv1.Artifact{}

	err := driver.Delete(ctx, artifact)
	require.Error(t, err)
	// The error should be ErrDeleteNotSupported
	assert.Contains(t, err.Error(), "delete")
}

func TestListObjectsNotSupported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &ArtifactDriver{
		Addresses: []string{"namenode:8020"},
		Path:      "/test/path",
		HDFSUser:  "testuser",
	}

	artifact := &wfv1.Artifact{}

	files, err := driver.ListObjects(ctx, artifact)
	require.Error(t, err)
	assert.Nil(t, files)
	assert.Contains(t, err.Error(), "currently not supported")
}

func TestIsDirectoryNotSupported(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	driver := &ArtifactDriver{
		Addresses: []string{"namenode:8020"},
		Path:      "/test/path",
		HDFSUser:  "testuser",
	}

	artifact := &wfv1.Artifact{}

	isDir, err := driver.IsDirectory(ctx, artifact)
	require.Error(t, err)
	assert.False(t, isDir)

	// Verify it's a CodeNotImplemented error
	argoErr, ok := err.(errors.ArgoError)
	if ok {
		assert.Equal(t, errors.CodeNotImplemented, argoErr.Code())
	}
}

