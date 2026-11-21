package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestIgnoredConfigWarnings(t *testing.T) {
	t.Run("NoWarningByDefault", func(t *testing.T) {
		assert.Empty(t, ignoredConfigWarnings(&config.Config{}))
	})
	t.Run("NoWarningForMetricsConfig", func(t *testing.T) {
		c := &config.Config{MetricsConfig: config.MetricsConfig{Path: "/metrics", Port: 9090}}
		assert.Empty(t, ignoredConfigWarnings(c))
	})
	t.Run("TelemetryConfigIsIgnored", func(t *testing.T) {
		for name, telemetryConfig := range map[string]config.MetricsConfig{
			"Path":    {Path: "/telemetry"},
			"Port":    {Port: 8081},
			"Enabled": {Enabled: new(false)},
		} {
			t.Run(name, func(t *testing.T) {
				warnings := ignoredConfigWarnings(&config.Config{TelemetryConfig: telemetryConfig}) //nolint:staticcheck // sets the deprecated field on purpose
				require.Len(t, warnings, 1)
				assert.Contains(t, warnings[0], "telemetryConfig")
				assert.Contains(t, warnings[0], "metricsConfig")
			})
		}
	})
}

func TestUpdateConfigWithIgnoredTelemetryConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.TelemetryConfig = config.MetricsConfig{Path: "/telemetry"} //nolint:staticcheck // sets the deprecated field on purpose
	require.NoError(t, controller.updateConfig(ctx))
}

func TestUpdateConfig(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo)
}
