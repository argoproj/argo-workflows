package http

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendHttpRequest(t *testing.T) {
	t.Run("SuccessfulRequest", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "http://www.google.com", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		response, err := SendHttpRequest(request)
		assert.NoError(t, err)
		assert.NotEmpty(t, response)
	})
	t.Run("NotFoundRequest", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "http://www.notfound.com/test", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		response, err := SendHttpRequest(request)
		assert.Error(t, err)
		assert.Empty(t, response)
		assert.Equal(t, "404 Not Found", err.Error())
	})
}
