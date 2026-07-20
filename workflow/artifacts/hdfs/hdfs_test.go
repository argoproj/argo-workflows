package hdfs

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// TestSaveStreamRemovesTempFileOnSaveError tests that when the delegated Save
// fails (unreachable HDFS), SaveStream returns the error and leaves no buffered
// temp file behind
func TestSaveStreamRemovesTempFileOnSaveError(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Isolate the driver's os.CreateTemp("", ...) into a per-test dir so the
	// leftover check does not race with other tests or with residue in shared /tmp.
	tmpDir := t.TempDir()
	t.Setenv("TMPDIR", tmpDir)

	driver := &ArtifactDriver{
		Addresses: []string{"nonexistent:8020"},
		Path:      "/test/path",
		HDFSUser:  "testuser",
	}

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

	err := driver.SaveStream(ctx, strings.NewReader("test content"), outputArtifact)
	require.Error(t, err, "Save must fail against an unreachable HDFS address")

	leftovers, globErr := filepath.Glob(filepath.Join(tmpDir, "hdfs-upload-*"))
	require.NoError(t, globErr)
	require.Empty(t, leftovers, "buffered temp file must be removed after SaveStream returns")
}
