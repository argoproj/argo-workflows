package kubeconfig

import (
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)


func Test_getDefaultTokenVersion(t *testing.T) {

	t.Run("No token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		os.Unsetenv("ARGO_TOKEN")
		assert.NoError(t, err)
		token, err := GetBearerToken(restConfig)
		assert.NoError(t, err)
		assert.Equal(t,restConfig.BearerToken, token)
	})
	t.Run("Env token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		restConfig.BearerToken="test"
		os.Setenv("ARGO_TOKEN", "test")
		token, err := GetBearerToken(restConfig)
		assert.NoError(t, err)
		assert.Equal(t,"test", token)
	})
	t.Run("RestConfig token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		os.Unsetenv("ARGO_TOKEN")
		assert.NoError(t, err)
		restConfig.BearerToken="test"
		token, err := GetBearerToken(restConfig)
		assert.NoError(t, err)
		assert.Equal(t,restConfig.BearerToken, token)
	})
}