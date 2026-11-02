package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateUploadedArtifactKey(t *testing.T) {
	const namespace = "my-ns"
	const validUUID = "12345678-1234-4234-8234-123456789012"

	t.Run("valid key", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns/"+validUUID+"/file.zip")
		require.NoError(t, err)
	})

	t.Run("wrong namespace prefix", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/other-ns/"+validUUID+"/file.zip")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with")
	})

	t.Run("missing uploads prefix", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "some/other/key/file.zip")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must start with")
	})

	t.Run("path traversal", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns/"+validUUID+"/../../../etc/passwd")
		require.Error(t, err)
	})

	t.Run("absolute path", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "/etc/passwd")
		require.Error(t, err)
	})

	t.Run("empty segment via double slash", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns//file.zip")
		require.Error(t, err)
	})

	t.Run("wrong segment count", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns/"+validUUID+"/sub/file.zip")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "4 segments")
	})

	t.Run("non-uuid segment", func(t *testing.T) {
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns/not-a-uuid/file.zip")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid UUID segment")
	})

	t.Run("uuid segment shaped like a hostname is still rejected", func(t *testing.T) {
		// Confirms the check is uuid.Parse, not merely "looks non-empty".
		err := ValidateUploadedArtifactKey(namespace, "uploads/my-ns/evil.example.com/file.zip")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid UUID segment")
	})
}
