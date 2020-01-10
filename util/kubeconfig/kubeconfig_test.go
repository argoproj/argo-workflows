package kubeconfig

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseToken(t *testing.T) {
	t.Run("Invalid", func(t *testing.T) {
		_, _, err := parseToken("")
		assert.Error(t, err)
	})
	t.Run("Valid", func(t *testing.T) {
		version, tokenBody, err := parseToken("v1:tokenBody")
		if assert.NoError(t, err){
			assert.Equal(t, 1, version)
			assert.Equal(t, "tokenBody", tokenBody)
		}
	})
}