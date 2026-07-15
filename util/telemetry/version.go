package telemetry

import (
	"context"

	"github.com/argoproj/argo-workflows/v4"
)

func AddVersion(ctx context.Context, m *Metrics) error {
	err := m.CreateBuiltinInstrument(InstrumentVersion)
	if err != nil {
		return err
	}

	version := argo.GetVersion()
	m.AddVersion(ctx, 1,
		version.Version,
		version.Platform,
		version.GoVersion,
		version.BuildDate,
		version.Compiler,
		version.GitCommit,
		version.GitTreeState,
		version.GitTag,
	)
	return nil
}
