package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"testing"

	"github.com/stretchr/testify/assert"
)


func Test_getDefaultTokenVersion(t *testing.T) {
	t.Run("No token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		token, err := GetBearerToken(restConfig)
		assert.NoError(t, err)
		assert.Equal(t,restConfig.BearerToken, token)
	})
	t.Run("token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		restConfig.BearerToken="Bearer"
		token, err := GetBearerToken(restConfig)
		assert.NoError(t, err)
		assert.Equal(t,restConfig.BearerToken, token)
	})
}