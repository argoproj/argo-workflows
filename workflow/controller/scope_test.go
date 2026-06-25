package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"

	wfv1 "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
)

func unsupportedArtifactSubPathResolution(t *testing.T, artifactString string) {
	ctx := logging.TestContext(t.Context())
	scope := createScope(nil)

	artifact := unmarshalArtifact(artifactString)

	scope.addArtifactToScope("steps.test.outputs.artifacts.art", *artifact)

	// Ensure that normal artifact resolution without adding subpaths works
	resolvedArtifact, err := scope.resolveArtifact(ctx, &wfv1.Artifact{From: "{{steps.test.outputs.artifacts.art}}"})
	require.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that we allow whitespaces in between names and brackets
	resolvedArtifact, err = scope.resolveArtifact(ctx, &wfv1.Artifact{From: "{{steps.test.outputs.artifacts.art}}"})
	require.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that adding a subpath results in an error being thrown
	_, err = scope.resolveArtifact(ctx, &wfv1.Artifact{SubPath: "some/subkey", From: "{{steps.test.outputs.artifacts.art}}"})
	require.Error(t, err)
}

func artifactSubPathResolution(t *testing.T, artifactString string, subPathArtifactString string) {
	ctx := logging.TestContext(t.Context())
	scope := createScope(nil)

	artifact := unmarshalArtifact(artifactString)
	originalArtifact := artifact.DeepCopy()
	artifactWithSubPath := unmarshalArtifact(subPathArtifactString)

	scope.addArtifactToScope("steps.test.outputs.artifacts.art", *artifact)

	// Ensure that normal artifact resolution without adding subpaths works
	resolvedArtifact, err := scope.resolveArtifact(ctx, &wfv1.Artifact{From: "{{steps.test.outputs.artifacts.art}}"})
	require.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifact)

	// Ensure that adding a subpath results in artifact key being modified
	resolvedArtifact, err = scope.resolveArtifact(ctx, &wfv1.Artifact{SubPath: "some/subkey", From: "{{steps.test.outputs.artifacts.art}}"})
	require.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)

	// Ensure that subpath template values are also resolved
	scope.addParamToScope("steps.test.outputs.parameters.subkey", "some")

	resolvedArtifact, err = scope.resolveArtifact(ctx, &wfv1.Artifact{SubPath: "{{steps.test.outputs.parameters.subkey}}/subkey", From: "{{steps.test.outputs.artifacts.art}}"})
	require.NoError(t, err)
	assert.Equal(t, resolvedArtifact, artifactWithSubPath)
	assert.Equal(t, artifact, originalArtifact)
}

func TestSubPathResolution(t *testing.T) {
	s3Artifact := `
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

	s3ArtifactWithSubpath := `
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

	ArtifactoryArtifact := `
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

	ArtifactoryArtifactWithSubpath := `
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

	GCSArtifact := `
  name: gcs-artifact
  path: some/local/path
  gcs:
    bucket: test-bucket
    key: /path/to/some/key
    serviceAccountKeySecret:
      key: accesskey
      name: my-gcs-cred
  `

	GCSArtifactWithSubpath := `
  name: gcs-artifact
  path: some/local/path
  gcs:
    bucket: test-bucket
    key: /path/to/some/key/some/subkey
    serviceAccountKeySecret:
      key: accesskey
      name: my-gcs-cred
  `

	HDFSArtifact := `
  name: hdfs-artifact
  path: some/local/path
  hdfs:
    addresses:
    - my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
    - my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
    path: /path/to/some/key
    hdfsUser: root
  `
	HDFSArtifactWithSubpath := `
  name: hdfs-artifact
  path: some/local/path
  hdfs:
    addresses:
    - my-hdfs-namenode-0.my-hdfs-namenode.default.svc.cluster.local:8020
    - my-hdfs-namenode-1.my-hdfs-namenode.default.svc.cluster.local:8020
    path: /path/to/some/key/some/subkey
    hdfsUser: root
  `

	OSSArtifact := `
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
	OSSArtifactWithSubpath := `
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

	HTTPArtifact := `
  name: oss-artifact
  path: some/local/path
  http:
    url: https://example.com
  `
	HTTPArtifactWithSubpath := `
  name: oss-artifact
  path: some/local/path
  http:
    url: https://example.com/some/subkey
  `

	GitArtifact := `
  name: git-artifact
  path: some/local/path
  git:
    repo: https://github.com/argoproj/argo-workflows
  `

	RawArtifact := `
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

func TestResolveParameters(t *testing.T) {
	assert := assert.New(t)
	tmpl := wfv1.Template{
		Name: "test",
		Inputs: wfv1.Inputs{
			Parameters: []wfv1.Parameter{
				{
					Name:  "one",
					Value: wfv1.AnyStringPtr(1),
				},
				{
					Name:  "two",
					Value: wfv1.AnyStringPtr(2),
				},
			},
			Artifacts: nil,
		},
	}

	scope := createScope(&tmpl)
	scope.addParamToScope("steps.t1.outputs.parameters.result", "4")
	scope.addParamToScope("workflows.arguments.param", "head")
	scope.addParamToScope("steps.coin-flip.outputs.parameters.result", "5")

	valFrom := &wfv1.ValueFrom{
		Expression: "inputs.parameters.one == '1' ? inputs.parameters.two: steps.t1.outputs.parameters.result",
	}
	result, _, err := scope.resolveParameter(valFrom)
	require.NoError(t, err)
	assert.Equal("2", result)

	valFrom = &wfv1.ValueFrom{
		Parameter: "{{steps.t1.outputs.parameters.result}}",
	}
	result, _, err = scope.resolveParameter(valFrom)
	require.NoError(t, err)
	assert.Equal("4", result)

	valFrom = &wfv1.ValueFrom{
		Expression: "inputs.parameters.one == 2 ? steps.t1.outputs.parameters.result :workflows.arguments.param",
	}
	result, _, err = scope.resolveParameter(valFrom)
	require.NoError(t, err)
	assert.Equal("head", result)

	valFrom = &wfv1.ValueFrom{
		Expression: "asInt(inputs.parameters.one) == 1 ? steps['coin-flip'].outputs.parameters.result :workflows.arguments.param",
	}
	result, _, err = scope.resolveParameter(valFrom)
	require.NoError(t, err)
	assert.Equal("5", result)

	valFrom = &wfv1.ValueFrom{
		Expression: "asInt(inputs.parameters.one) == 1 ? steps[\"coin-flip\"].outputs.parameters.result :workflows.arguments.param",
	}
	result, _, err = scope.resolveParameter(valFrom)
	require.NoError(t, err)
	assert.Equal("5", result)
}

// TestSkippedOptionalExpressionResolution verifies the *string/nil-for-optional model: a skipped
// output (no producer default) is stored as nil, so a ValueFrom.Expression can fall back via `??`,
// while a legitimately empty output ("") is NOT treated as absent.
func TestSkippedOptionalExpressionResolution(t *testing.T) {
	const ref = "tasks.producer.outputs.parameters.msg"
	const expr = "tasks.producer.outputs.parameters.msg ?? 'fallback-from-expr'"

	t.Run("skipped resolves to fallback", func(t *testing.T) {
		scope := createScope(nil)
		scope.addSkippedParamToScope(ref) // skipped, no producer default -> nil
		val, _, err := scope.resolveParameter(&wfv1.ValueFrom{Expression: expr})
		require.NoError(t, err)
		assert.Equal(t, "fallback-from-expr", val)
	})

	t.Run("real empty value is preserved", func(t *testing.T) {
		scope := createScope(nil)
		scope.addParamToScope(ref, "") // produced an empty string -> NOT absent
		val, _, err := scope.resolveParameter(&wfv1.ValueFrom{Expression: expr})
		require.NoError(t, err)
		assert.Empty(t, val)
	})

	t.Run("unhandled nil errors like inline expressions", func(t *testing.T) {
		scope := createScope(nil)
		scope.addSkippedParamToScope(ref) // skipped, no producer default -> nil
		_, _, err := scope.resolveParameter(&wfv1.ValueFrom{Expression: ref})
		require.ErrorContains(t, err, "failed to evaluate expression")
	})
}

// TestAbsentOptionalRefRequiresTag verifies that absentOptionalRef only matches a real pure
// "{{...}}" reference: a literal argument value that merely spells out a scope key is data, not a
// reference, and must not be treated as a skipped-output ref (which would get the argument silently
// replaced with the absent-optional sentinel); composite and nested values are not pure references either.
func TestAbsentOptionalRefRequiresTag(t *testing.T) {
	const ref = "tasks.producer.outputs.parameters.msg"
	scope := createScope(nil)
	scope.addSkippedParamToScope(ref)
	scope.addParamToScope("tasks.real.outputs.parameters.msg", "value")

	assert.True(t, scope.absentOptionalRef("{{"+ref+"}}"), "a pure tag referencing the skipped output should match")
	assert.True(t, scope.absentOptionalRef("{{ "+ref+" }}"), "inner whitespace is trimmed like simple-tag resolution")
	assert.False(t, scope.absentOptionalRef(ref), "a brace-less literal equal to a scope key is data, not a reference")
	assert.False(t, scope.absentOptionalRef("x-{{"+ref+"}}-y"), "composite values are not pure references")
	assert.False(t, scope.absentOptionalRef("{{outer-{{"+ref+"}}}}"), "nested tags are not pure references")
	assert.False(t, scope.absentOptionalRef("{{tasks.real.outputs.parameters.msg}}"), "a reference to a produced value is not absent")
	assert.False(t, scope.absentOptionalRef("{{tasks.unknown.outputs.parameters.msg}}"), "an unknown key is unresolved, not absent")
}
