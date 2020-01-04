package kubeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetBearerToken(t *testing.T) {
	t.SkipNow()
	config, err := DefaultRestConfig()
	if assert.NoError(t, err) {
		token, err := GetBearerToken(config)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		restConfig, err := GetRestConfig(token)
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Equal(t, restConfig.Host, config.Host)
		assert.Equal(t, restConfig.APIPath, config.APIPath)
		assert.Equal(t, restConfig.ServerName, config.ServerName)
		assert.Equal(t, restConfig.ContentConfig, config.ContentConfig)
		assert.Equal(t, restConfig.Username, config.Username)
		assert.Equal(t, restConfig.Password, config.Password)
		assert.Equal(t, restConfig.BearerToken, config.BearerToken)
		assert.Empty(t, restConfig.BearerTokenFile)
		assert.Equal(t, restConfig.Impersonate, config.Impersonate)
		assert.Equal(t, restConfig.AuthProvider, config.AuthProvider)
		assert.Empty(t, restConfig.ExecProvider)
		assert.Equal(t, restConfig.UserAgent, config.UserAgent)
		assert.Equal(t, restConfig.QPS, config.QPS)
		assert.Equal(t, restConfig.Burst, config.Burst)
		assert.Equal(t, restConfig.Timeout, config.Timeout)
		assert.Equal(t, restConfig.TLSClientConfig.ServerName, config.TLSClientConfig.ServerName)
		assert.Empty(t, restConfig.TLSClientConfig.CAFile)
		assert.Empty(t, restConfig.TLSClientConfig.CertFile)
		assert.Empty(t, restConfig.TLSClientConfig.KeyFile)
	}
}
