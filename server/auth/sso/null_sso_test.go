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
	assert.Equal(t, http.StatusNotImplemented, recorder.Result().StatusCode)
}

func Test_nullSSO_HandleRedirect(t *testing.T) {
	recorder := httptest.NewRecorder()
	NullSSO.HandleRedirect(recorder, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, recorder.Result().StatusCode)
}
