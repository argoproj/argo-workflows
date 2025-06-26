package controller

import (
	"context"
	"testing"

	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateConfig(t *testing.T) {
	cancel, controller := newController()
	defer cancel()

	log := logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	ctx := logging.WithLogger(context.Background(), log)

	err := controller.updateConfig(ctx)
	require.NoError(t, err)
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo)
}
