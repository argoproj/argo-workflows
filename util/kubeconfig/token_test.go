package kubeconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getDefaultTokenVersion(t *testing.T) {
	t.Run("v2", func(t *testing.T) {
		_ = os.Setenv("ARGO_TOKEN_VERSION", "v2")
		defer func() { _ = os.Unsetenv("ARGO_TOKEN_VERSION") }()
		_ = os.Setenv("ARGO_V2_TOKEN", "token")
		defer func() { _ = os.Unsetenv("ARGO_V2_TOKEN") }()

		assert.Equal(t, tokenVersion2, getDefaultTokenVersion())
		token, err := GetBearerToken(nil)
		if assert.NoError(t, err) {
			assert.Equal(t, "v2:token", token)
		}
	})
}

func Test_getV2TokenBody(t *testing.T) {
	t.Run("Undefined", func(t *testing.T) {
		_, err := getV2TokenBody()
		assert.Error(t, err)
	})
	t.Run("Defined", func(t *testing.T) {
		_ = os.Setenv("ARGO_V2_TOKEN", "token")
		defer func() { _ = os.Unsetenv("ARGO_V2_TOKEN") }()
		token, err := getV2TokenBody()
		if assert.NoError(t, err) {
			assert.Equal(t, "token", token)
		}
	})
}
