package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func unsupportedArtifactSubPathResolution(t *testing.T, artifactString string) {
	scope := wfScope{
		tmpl:  nil,
		scope: make(map[string]interface{}),
	}

	artifact := unmarshalArtifact(artifactString)

	scope.addArtifactToScope("steps.test.outputs.artifacts.art", *artifact)

	// Ensure that normal artifact resolution without adding subpaths works
	resolvedArtifact, err := scope.resolveArtifact("{{steps.test.outputs.artifacts.art}}", "")
	assert.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that adding a subpath results in an error being thrown
	_, err = scope.resolveArtifact("{{steps.test.outputs.artifacts.art}}", "some/subkey")
	assert.Error(t, err)
}

func artifactSubPathResolution(t *testing.T, artifactString string, subPathArtifactString string) {
	scope := wfScope{
		tmpl:  nil,
		scope: make(map[string]interface{}),
	}

	artifact := unmarshalArtifact(artifactString)
	originalArtifact := artifact.DeepCopy()
	artifactWithSubPath := unmarshalArtifact(subPathArtifactString)

	scope.addArtifactToScope("steps.test.outputs.artifacts.art", *artifact)

	// Ensure that normal artifact resolution without adding subpaths works
	resolvedArtifact, err := scope.resolveArtifact("{{steps.test.outputs.artifacts.art}}", "")
	assert.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that adding a subpath results in artifact key being modified
	resolvedArtifact, err = scope.resolveArtifact("{{steps.test.outputs.artifacts.art}}", "some/subkey")
	assert.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)

	// Ensure that subpath template values are also resolved
	scope.addParamToScope("steps.test.outputs.parameters.subkey", "some")

	resolvedArtifact, err = scope.resolveArtifact("{{steps.test.outputs.artifacts.art}}", "{{steps.test.outputs.parameters.subkey}}/subkey")
	assert.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)
	assert.Equal(t, artifact, originalArtifact)
}

func TestSubPathResolution(t *testing.T) {
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
  name: gcs-artifact
  path: some/local/path
  gcs:
    bucket: test-bucket
    key: /path/to/some/key
    serviceAccountKeySecret:
      key: accesskey
      name: my-gcs-cred
  `

	var GCSArtifactWithSubpath = `
  name: gcs-artifact
  path: some/local/path
  gcs:
    bucket: test-bucket
    key: /path/to/some/key/some/subkey
    serviceAccountKeySecret:
      key: accesskey
      name: my-gcs-cred
  `

	var HDFSArtifact = `
  name: hdfs-artifact
  path: some/local/path
  hdfs:
    addresses:
    - my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
    - my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
    path: /path/to/some/key
    hdfsUser: root
  `
	var HDFSArtifactWithSubpath = `
  name: hdfs-artifact
  path: some/local/path
  hdfs:
    addresses:
    - my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
    - my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
    path: /path/to/some/key/some/subkey
    hdfsUser: root
  `

	var OSSArtifact = `
  name: oss-artifact
  path: some/local/path
  oss:
    endpoint: http://oss-cn-hangzhou-zmf.aliyuncs.com
    bucket: test-bucket-name
    key: path/to/some/key
    accessKeySecret:
      name: my-oss-credentials
      key: accessKey
    secretKeySecret:
      name: my-oss-credentials
      key: secretKey
  `
	var OSSArtifactWithSubpath = `
  name: oss-artifact
  path: some/local/path
  oss:
    endpoint: http://oss-cn-hangzhou-zmf.aliyuncs.com
    bucket: test-bucket-name
    key: path/to/some/key/some/subkey
    accessKeySecret:
      name: my-oss-credentials
      key: accessKey
    secretKeySecret:
      name: my-oss-credentials
      key: secretKey
  `

	var HTTPArtifact = `
  name: oss-artifact
  path: some/local/path
  http:
    url: https://example.com
  `
	var HTTPArtifactWithSubpath = `
  name: oss-artifact
  path: some/local/path
  http:
    url: https://example.com/some/subkey
  `

	var GitArtifact = `
  name: git-artifact
  path: some/local/path
  git:
    repo: https://github.com/argoproj/argo
  `

	var RawArtifact = `
  name: raw-artifact
  path: some/local/path
  raw:
    data: some-long-artifact-data-string
  `

	t.Run("S3 Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, s3Artifact, s3ArtifactWithSubpath)
	})
	t.Run("Artifactory Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, ArtifactoryArtifact, ArtifactoryArtifactWithSubpath)
	})
	t.Run("GCS Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, GCSArtifact, GCSArtifactWithSubpath)
	})
	t.Run("HDFS Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, HDFSArtifact, HDFSArtifactWithSubpath)
	})
	t.Run("OSS Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, OSSArtifact, OSSArtifactWithSubpath)
	})
	t.Run("HTTP Artifact SubPath Resolution", func(t *testing.T) {
		artifactSubPathResolution(t, HTTPArtifact, HTTPArtifactWithSubpath)
	})

	t.Run("Git Artifact SubPath Unsupported Resolution", func(t *testing.T) {
		unsupportedArtifactSubPathResolution(t, GitArtifact)
	})
	t.Run("Raw Artifact SubPath Unsupported Resolution", func(t *testing.T) {
		unsupportedArtifactSubPathResolution(t, RawArtifact)
	})
}
