package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var s3Artifact = `
name: s3-artifact
path: some/local/path
s3:
  endpoint: minio:9000
  bucket: test-bucket
  key: /path/to/some/key
  accessKeySecret:
    key: accesskey
    name: my-minio-cred
  secretKeySecret:
    key: secretkey
    name: my-minio-cred
`

var s3ArtifactWithSubpath = `
name: s3-artifact
path: some/local/path
s3:
  endpoint: minio:9000
  bucket: test-bucket
  key: /path/to/some/key/some/subkey
  accessKeySecret:
    key: accesskey
    name: my-minio-cred
  secretKeySecret:
    key: secretkey
    name: my-minio-cred
`

var ArtifactoryArtifact = `
name: artifactory-artifact
path: some/local/path
artifactory:
  url: https://artifactory.example.com/some/local/path
  usernameSecret:
    key: accesskey
    name: my-artifactory-cred
  passwordSecret:
    key: secretkey
    name: my-artifactory-cred
`

var ArtifactoryArtifactWithSubpath = `
name: artifactory-artifact
path: some/local/path
artifactory:
  url: https://artifactory.example.com/some/local/path/some/subkey
  usernameSecret:
    key: accesskey
    name: my-artifactory-cred
  passwordSecret:
    key: secretkey
    name: my-artifactory-cred
`

var GCSArtifact = `
name: s3-artifact
path: some/local/path
s3:
  bucket: test-bucket
  key: /path/to/some/key
  serviceAccountKeySecret:
    key: accesskey
    name: my-gcs-cred
`

var GCSArtifactWithSubpath = `
name: s3-artifact
path: some/local/path
s3:
  bucket: test-bucket
  key: /path/to/some/key/some/subkey
  serviceAccountKeySecret:
    key: accesskey
    name: my-gcs-cred
`

func artifactSubPathResolution(t *testing.T, artifactString string, subPathArtifactString string) {
	scope := wfScope{
		tmpl:  nil,
		scope: make(map[string]interface{}),
	}

	artifact := unmarshalArtifact(artifactString)
	originalArtifact := artifact.DeepCopy()
	artifactWithSubPath := unmarshalArtifact(subPathArtifactString)

	scope.addArtifactToScope("steps.test", *artifact)

	// Ensure that normal artifact resolution without adding subpaths works
	resolvedArtifact, err := scope.resolveArtifact("steps.test", "")
	assert.Nil(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that adding a subpath results in artifact key being modified
	resolvedArtifact, err = scope.resolveArtifact("steps.test", "some/subkey")
	assert.Nil(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)

	// Ensure that resolution with subpath operation does not overwrite the original artifact
	resolvedArtifact, err = scope.resolveArtifact("steps.test", "some/subkey")
	assert.Nil(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)
	assert.Equal(t, artifact, originalArtifact)
}

func TestSubPathResolution(t *testing.T) {
	t.Run("S3 Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, s3Artifact, s3ArtifactWithSubpath)
	})
	t.Run("Artifactory Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, ArtifactoryArtifact, ArtifactoryArtifactWithSubpath)
	})
	t.Run("GCS Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, GCSArtifact, GCSArtifactWithSubpath)
	})
}
