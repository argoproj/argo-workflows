package kubeconfig

import (
	"io/ioutil"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

const bearerToken = "bearertoken"

const config = `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:6443
  name: k3s-default
contexts:
- context:
    cluster: k3s-default
    namespace: argo
    user: k3s-default
  name: k3s-default
current-context: k3s-default
kind: Config
preferences: {}
users:
- name: k3s-default
  user:
    password: admin
    username: admin
`

func Test_BasicAuthString(t *testing.T) {

	t.Run("Basic Auth", func(t *testing.T) {

		restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(config))
		authString, err := GetAuthString(restConfig)
		assert.NoError(t, err)
		assert.True(t, IsBasicAuthScheme(authString))
		token := strings.TrimPrefix(authString, BasicAuthScheme)
		uname, pwd, ok := decodeBasicAuthToken(token)
		if assert.True(t, ok) {
			assert.Equal(t, "admin", uname)
			assert.Equal(t, "admin", pwd)
		}
		file, err := ioutil.TempFile("", "config.yaml")
		file.WriteString(config)
		file.Close()
		os.Setenv("KUBECONFIG", file.Name())
		config, err := GetRestConfig(authString)
		if assert.NoError(t, err) {
			assert.Equal(t, "admin", config.Username)
			assert.Equal(t, "admin", config.Password)
		}

	})
}

func Test_BearerAuthString(t *testing.T) {

	restConfig := rest.Config{}

	t.Run("Bearer Auth", func(t *testing.T) {
		os.Setenv("ARGO_TOKEN", bearerToken)
		//_, err := ioutil.TempDir("~",".kube")

		authString, err := GetAuthString(&restConfig)
		assert.NoError(t, err)
		assert.True(t, IsBearerAuthScheme(authString))
		config, err := GetRestConfig(authString)
		assert.NoError(t, err)
		if assert.NoError(t, err) {
			assert.Equal(t, bearerToken, config.BearerToken)
		}
	})
}
