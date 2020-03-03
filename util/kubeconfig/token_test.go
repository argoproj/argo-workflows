package kubeconfig

import (
	"os"
	"testing"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/stretchr/testify/assert"
)

func Test_getDefaultTokenVersion(t *testing.T) {

	t.Run("No token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		restConfig.BearerToken = ""
		envToken := os.Getenv("ARGO_TOKEN")
		os.Unsetenv("ARGO_TOKEN")
		defer os.Setenv("ARGO_TOKEN", envToken)
		_, err = GetBearerToken(restConfig, "")
		assert.Error(t, err)
	})
	t.Run("Existing token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		restConfig.BearerToken = "test12"
		envToken := os.Getenv("ARGO_TOKEN")
		os.Setenv("ARGO_TOKEN", "test")
		defer os.Setenv("ARGO_TOKEN", envToken)
		token, err := GetBearerToken(restConfig, "")
		assert.NoError(t, err)
		assert.Equal(t, "test12", token)
	})
	t.Run("Env token", func(t *testing.T) {
		restConfig, err := clientcmd.DefaultClientConfig.ClientConfig()
		assert.NoError(t, err)
		restConfig.BearerToken = ""
		envToken := os.Getenv("ARGO_TOKEN")
		os.Setenv("ARGO_TOKEN", "test")
		defer os.Setenv("ARGO_TOKEN", envToken)
		token, err := GetBearerToken(restConfig, "")
		assert.NoError(t, err)
		assert.Equal(t, "test", token)
	})
}
