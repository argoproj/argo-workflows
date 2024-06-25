package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"

	"github.com/argoproj/argo-workflows/v3"
)

func TestBuildInfo(t *testing.T) {
	_, te, err := CreateDefaultTestMetrics()
	if assert.NoError(t, err) {
		assert.NotNil(t, te)
		version := argo.GetVersion()
		attribs := attribute.NewSet(
			attribute.String(labelBuildVersion, version.Version),
			attribute.String(labelBuildPlatform, version.Platform),
			attribute.String(labelBuildGoVer, version.GoVersion),
			attribute.String(labelBuildDate, version.BuildDate),
			attribute.String(labelBuildCompiler, version.Compiler),
			attribute.String(labelBuildGitCommit, version.GitCommit),
			attribute.String(labelBuildGitTreeState, version.GitTreeState),
			attribute.String(labelBuildGitTag, version.GitTag),
		)
		val, err := te.GetInt64CounterValue(`controller_build_info`, &attribs)
		if assert.NoError(t, err) {
			assert.Equal(t, int64(1), val)
		}
	}
}
