package sso

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrypto(t *testing.T) {
	key, err := generateKey()
	if assert.NoError(t, err) {
		assert.Len(t, key, 32, "is 256 bits")
	}
	data, err := encrypt(key, []byte("my-string"))
	if assert.NoError(t, err) {
		assert.NotEmpty(t, data)
	}
	data, err = decrypt(key, data)
	if assert.NoError(t, err) {
		assert.Equal(t, []byte("my-string"), data)
	}
}
