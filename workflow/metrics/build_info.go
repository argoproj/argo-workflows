package metrics

import (
	"context"

	"github.com/argoproj/argo-workflows/v3"
)

func addBuildInfo(ctx context.Context, m *Metrics) error {
	const nameBuildInfo = `controller_build_info`
	err := m.createInstrument(int64Counter,
		nameBuildInfo,
		"Build Information for the Argo Workflows Controller",
		"{unused}",
		withAsBuiltIn(),
	)
	if err != nil {
		return err
	}

	version := argo.GetVersion()
	m.addInt(ctx, nameBuildInfo, 1, instAttribs{
		{name: labelBuildVersion, value: version.Version},
		{name: labelBuildPlatform, value: version.Platform},
		{name: labelBuildGoVer, value: version.GoVersion},
		{name: labelBuildDate, value: version.BuildDate},
		{name: labelBuildCompiler, value: version.Compiler},
		{name: labelBuildGitCommit, value: version.GitCommit},
		{name: labelBuildGitTreeState, value: version.GitTreeState},
		{name: labelBuildGitTag, value: version.GitTag},
	})
	return nil
}
