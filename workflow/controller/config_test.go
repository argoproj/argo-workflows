package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

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

func TestUpdateConfigRejectsInvalidHotWorkers(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	cancel, controller := newController(ctx)
	defer cancel()

	controller.Config.HotWorkers = &config.HotWorkersConfig{Enabled: true, ReleaseReserveSeconds: new(int64(30))}
	require.EqualError(t, controller.updateConfig(ctx), "hotWorkers.releaseReserveSeconds (30) must be less than hotWorkers.terminationGracePeriodSeconds (30)")
}
