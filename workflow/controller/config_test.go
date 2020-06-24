package controller

import (
	"k8s.io/client-go/util/workqueue"
	"testing"

	"github.com/argoproj/argo/config"
	"github.com/stretchr/testify/assert"
)

func TestUpdateConfig(t *testing.T) {
	cancel, controller := newController()
	defer cancel()
	controller.throttler = NewThrottler(0, workqueue.NewNamedRateLimitingQueue(nil, ""))
	err := controller.updateConfig(config.Config{ExecutorImage: "argoexec:latest", })
	assert.NoError(t, err)
	assert.NotNil(t, controller.Config)
	assert.NotNil(t, controller.archiveLabelSelector)
	assert.NotNil(t, controller.wfArchive)
	assert.NotNil(t, controller.offloadNodeStatusRepo )
}
