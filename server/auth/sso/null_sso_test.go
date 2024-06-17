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
	w := httptest.NewRecorder()
	NullSSO.HandleCallback(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}

func Test_nullSSO_HandleRedirect(t *testing.T) {
	w := httptest.NewRecorder()
	NullSSO.HandleRedirect(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.Code)
}
