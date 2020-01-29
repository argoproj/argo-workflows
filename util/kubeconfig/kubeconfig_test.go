package kubeconfig

import (

	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

const username = "admin"
const password = "admin"

const bearerToken = "bearertoken"

func Test_BasicAuthString(t *testing.T) {
	restConfig := rest.Config{}

	restConfig.Username = username
	restConfig.Password = password

	t.Run("Basic Auth", func(t *testing.T) {
		authString, err := GetAuthString(&restConfig)
		assert.NoError(t, err)
		outConfig, err := GetRestConfig(authString)
		if assert.NoError(t, err) {
			assert.Equal(t, outConfig.Username, username)
			assert.Equal(t, outConfig.Password, password)
		}
	})
}

func Test_BearerAuthString(t *testing.T) {

	restConfig := rest.Config{}

	t.Run("Bearer Auth", func(t *testing.T) {
		os.Setenv("ARGO_TOKEN", bearerToken)
		authString, err := GetAuthString(&restConfig)
		assert.NoError(t, err)
		outConfig, err := GetRestConfig(authString)
		if assert.NoError(t, err) {
			assert.Equal(t, outConfig.BearerToken, bearerToken)
		}
	})
}
