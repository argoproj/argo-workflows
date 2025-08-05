package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v3"
)

func TestVersion(t *testing.T) {
	_, te, err := createDefaultTestMetrics()
	require.NoError(t, err)
	assert.NotNil(t, te)
	version := argo.GetVersion()
	attribs := attribute.NewSet(
		attribute.String(AttribBuildVersion, version.Version),
		attribute.String(AttribBuildPlatform, version.Platform),
		attribute.String(AttribBuildGoVersion, version.GoVersion),
		attribute.String(AttribBuildDate, version.BuildDate),
		attribute.String(AttribBuildCompiler, version.Compiler),
		attribute.String(AttribBuildGitCommit, version.GitCommit),
		attribute.String(AttribBuildGitTreeState, version.GitTreeState),
		attribute.String(AttribBuildGitTag, version.GitTag),
	)
	val, err := te.GetInt64CounterValue(`version`, &attribs)
	require.NoError(t, err)
	assert.Equal(t, int64(1), val)
}
