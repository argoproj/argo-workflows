package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookup(t *testing.T) {
	t.Run("argoproj/argosay:v1", func(t *testing.T) {
		v, err := Lookup("argoproj/argosay:v1")
		assert.NoError(t, err)
		assert.Equal(t, []string{"cowsay"}, v)
	})
	t.Run("argoproj/argosay:v2", func(t *testing.T) {
		v, err := Lookup("argoproj/argosay:v2")
		assert.NoError(t, err)
		assert.Equal(t, []string{"/argosay"}, v)
	})
	t.Run("docker/whalesay:latest", func(t *testing.T) {
		_, err := Lookup("docker/whalesay:latest")
		assert.Error(t, err)
	})
	t.Run("python:alpine3.6", func(t *testing.T) {
		v, err := Lookup("python:alpine3.6")
		assert.NoError(t, err)
		assert.Equal(t, []string{"python3"}, v)
	})
}
