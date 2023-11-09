package serviceaccount

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

// sub = 1234567890
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c" //nolint:gosec

func TestClaimSetFor(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{})
		if assert.NoError(t, err) {
			assert.Nil(t, claims)
		}
	})
	t.Run("Basic", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{Username: "my-username"})
		if assert.NoError(t, err) {
			assert.Empty(t, claims.Issuer)
			assert.Equal(t, "my-username", claims.Subject)
		}
	})
	t.Run("BadBearerToken", func(t *testing.T) {
		_, err := ClaimSetFor(&rest.Config{BearerToken: "bad"})
		assert.Error(t, err)
	})
	t.Run("BearerToken", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{BearerToken: token})
		if assert.NoError(t, err) {
			assert.Empty(t, claims.Issuer)
			assert.Equal(t, "1234567890", claims.Subject)
		}
	})

	// set-up test
	tmp, err := os.CreateTemp("", "")
	assert.NoError(t, err)
	err = os.WriteFile(tmp.Name(), []byte(token), 0o600)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()

	t.Run("BearerTokenFile", func(t *testing.T) {
		claims, err := ClaimSetFor(&rest.Config{BearerTokenFile: tmp.Name()})
		if assert.NoError(t, err) {
			assert.Empty(t, claims.Issuer)
			assert.Equal(t, "1234567890", claims.Subject)
		}
	})
}
