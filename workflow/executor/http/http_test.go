package http

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/utils/pointer"
)

func TestSendHttpRequest(t *testing.T) {
	t.Run("SuccessfulRequest", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "http://httpstat.us/200", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		_, err = SendHttpRequest(request, nil)
		assert.NoError(t, err)
	})
	t.Run("NotFoundRequest", func(t *testing.T) {
		request, err := http.NewRequest(http.MethodGet, "http://httpstat.us/404", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		response, err := SendHttpRequest(request, nil)
		assert.Error(t, err)
		assert.Empty(t, response)
		assert.Equal(t, "404 Not Found", err.Error())
	})
	t.Run("TimeoutRequest", func(t *testing.T) {
		// Request sleeps for 4 seconds, but timeout is 2
		request, err := http.NewRequest(http.MethodGet, "http://httpstat.us/200?sleep=4000", bytes.NewBuffer([]byte{}))
		assert.NoError(t, err)
		response, err := SendHttpRequest(request, pointer.Int64Ptr(2))
		assert.Error(t, err)
		assert.Empty(t, response)
		assert.Equal(t, `Get "http://httpstat.us/200?sleep=4000": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`, err.Error())
	})
}
