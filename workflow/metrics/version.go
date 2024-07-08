package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3"
)

func addVersion(ctx context.Context, m *Metrics) error {
	const nameVersion = `version`
	err := m.createInstrument(int64Counter,
		nameVersion,
		"Build metadata for this Controller",
		"{unused}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	version := argo.GetVersion()
	m.addInt(ctx, nameVersion, 1, instAttribs{
		{name: labelBuildVersion, value: version.Version},
		{name: labelBuildPlatform, value: version.Platform},
		{name: labelBuildGoVersion, value: version.GoVersion},
		{name: labelBuildDate, value: version.BuildDate},
		{name: labelBuildCompiler, value: version.Compiler},
		{name: labelBuildGitCommit, value: version.GitCommit},
		{name: labelBuildGitTreeState, value: version.GitTreeState},
		{name: labelBuildGitTag, value: version.GitTag},
	})
	return nil
}
