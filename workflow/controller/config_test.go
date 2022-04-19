package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/config"
)

func TestUpdateConfig(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	err := controller.updateConfig(&config.Config{})
	assert.NoError(t, err)
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo)
}
