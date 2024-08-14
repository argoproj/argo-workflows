package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateConfig(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	err := controller.updateConfig()
	require.NoError(t, err)
	require.NotNil(t, controller.Config)
	require.NotNil(t, controller.archiveLabelSelector)
	require.NotNil(t, controller.wfArchive)
	require.NotNil(t, controller.offloadNodeStatusRepo)
}
