package sso

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_nullSSO_Authorize(t *testing.T) {
	_, err := NullSSO.Authorize("")
	assert.Error(t, err)
}

func Test_nullSSO_HandleCallback(t *testing.T) {
	recorder := httptest.NewRecorder()
	NullSSO.HandleCallback(recorder, &http.Request{})
	result := recorder.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusNotImplemented, result.StatusCode)
}

func Test_nullSSO_HandleRedirect(t *testing.T) {
	recorder := httptest.NewRecorder()
	NullSSO.HandleRedirect(recorder, &http.Request{})
	result := recorder.Result()
	defer result.Body.Close()
	assert.Equal(t, http.StatusNotImplemented, result.StatusCode)
}
