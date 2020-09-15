package userinfo

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
)

// sub = 1234567890
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"

func TestUserInfoFor(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		claimSet, err := UserInfoFor(&rest.Config{})
		if assert.NoError(t, err) {
			assert.Nil(t, claimSet)
		}
	})
	t.Run("Basic", func(t *testing.T) {
		claimSet, err := UserInfoFor(&rest.Config{Username: "my-username"})
		if assert.NoError(t, err) {
			assert.Equal(t, "my-username", claimSet.Subject)
		}
	})
	t.Run("BadBearerToken", func(t *testing.T) {
		_, err := UserInfoFor(&rest.Config{BearerToken: "bad"})
		assert.Error(t, err)
	})

	// set-up test
	tmp, err := ioutil.TempFile("", "")
	assert.NoError(t, err)
	err = ioutil.WriteFile(tmp.Name(), []byte(token), 0644)
	assert.NoError(t, err)
	defer func() { _ = os.Remove(tmp.Name()) }()

	t.Run("BearerTokenFile", func(t *testing.T) {
		claimSet, err := UserInfoFor(&rest.Config{BearerTokenFile: tmp.Name()})
		if assert.NoError(t, err) {
			assert.Equal(t, "1234567890", claimSet.Subject)
		}
	})
}
