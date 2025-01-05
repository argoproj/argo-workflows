package telemetry

import (
	"context"

	"github.com/argoproj/argo-workflows/v3"
)

func AddVersion(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(InstrumentVersion)
	if err != nil {
		return err
	}

	version := argo.GetVersion()
	m.AddInt(ctx, InstrumentVersion.Name(), 1, InstAttribs{
		{Name: AttribBuildVersion, Value: version.Version},
		{Name: AttribBuildPlatform, Value: version.Platform},
		{Name: AttribBuildGoVersion, Value: version.GoVersion},
		{Name: AttribBuildDate, Value: version.BuildDate},
		{Name: AttribBuildCompiler, Value: version.Compiler},
		{Name: AttribBuildGitCommit, Value: version.GitCommit},
		{Name: AttribBuildGitTreeState, Value: version.GitTreeState},
		{Name: AttribBuildGitTag, Value: version.GitTag},
	})
	return nil
}
