package gcs

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"

	argoErrors "github.com/argoproj/argo-workflows/v3/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

type tlsHandshakeTimeoutError struct{}

func (tlsHandshakeTimeoutError) Temporary() bool { return true }
func (tlsHandshakeTimeoutError) Error() string   { return "net/http: TLS handshake timeout" }

func TestIsTransientGCSErr(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	for _, test := range []struct {
		err         error
		shouldretry bool
	}{
		{&googleapi.Error{Code: 0}, false},
		{argoErrors.New(argoErrors.CodeNotFound, "no results for key: foo/bar"), false},
		{&googleapi.Error{Code: 429}, true},
		{&googleapi.Error{Code: 504}, true},
		{&googleapi.Error{Code: 518}, true},
		{&googleapi.Error{Code: 599}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("connection refused")}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("compute: Received 504 `Gateway Timeout\n`")}, true},
		{&url.Error{Op: "blah", URL: "blah", Err: errors.New("http2: client connection lost")}, true},
		{io.ErrUnexpectedEOF, true},
		{&tlsHandshakeTimeoutError{}, true},
		{fmt.Errorf("Test unwrapping of a temporary error: %w", &googleapi.Error{Code: 500}), true},
		{fmt.Errorf("Test unwrapping of a non-retriable error: %w", &googleapi.Error{Code: 400}), false},
		{fmt.Errorf("writer close: Post \"https://storage.googleapis.com/upload/storage/v1/b/bucket/o?alt=json&name=test.json&uploadType=multipart\": compute: Received 504 `Gateway Timeout\n`"), true},
		{fmt.Errorf("http2: client connection lost"), true},
	} {
		got := isTransientGCSErr(ctx, test.err)
		if got != test.shouldretry {
			t.Errorf("%+v: got %v, want %v", test, got, test.shouldretry)
		}
	}
}

// TestNewGCSClientWithCredentialError tests error handling when credentials are invalid
func TestNewGCSClientWithCredentialError(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Invalid JSON should cause an error
	_, err := newGCSClientWithCredential(ctx, "invalid-json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CredentialsFromJSON")
}

// TestListFileRelPaths tests the listFileRelPaths function
func TestListFileRelPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Create test directory structure
	subDir := filepath.Join(tempDir, "subdir")
	err := os.MkdirAll(subDir, 0755)
	require.NoError(t, err)

	// Create files
	err = os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("content1"), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0600)
	require.NoError(t, err)

	// Test listing
	files, err := listFileRelPaths(tempDir+string(os.PathSeparator), "")
	require.NoError(t, err)
	assert.Len(t, files, 2)

	// Check that both files are listed (order may vary)
	hasFile1 := false
	hasFile2 := false
	for _, f := range files {
		if f == "file1.txt" {
			hasFile1 = true
		}
		if f == "subdir"+string(os.PathSeparator)+"file2.txt" {
			hasFile2 = true
		}
	}
	assert.True(t, hasFile1, "file1.txt should be in the list")
	assert.True(t, hasFile2, "subdir/file2.txt should be in the list")
}

// TestListFileRelPathsEmptyDir tests listing an empty directory
func TestListFileRelPathsEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	files, err := listFileRelPaths(tempDir+string(os.PathSeparator), "")
	require.NoError(t, err)
	assert.Empty(t, files)
}

// TestListFileRelPathsNonExistent tests listing a non-existent directory
func TestListFileRelPathsNonExistent(t *testing.T) {
	_, err := listFileRelPaths("/non/existent/path/", "")
	require.Error(t, err)
}
