package sso

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func Test_nullSSO_Authorize(t *testing.T) {
	user, err := NullSSO.Authorize(context.Background(), "")
	if assert.Error(t, err) {
		assert.Equal(t, wfv1.NullUser, user)
	}
}

func Test_nullSSO_HandleCallback(t *testing.T) {
	w := &testhttp.TestResponseWriter{}
	NullSSO.HandleCallback(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.StatusCode)
}

func Test_nullSSO_HandleRedirect(t *testing.T) {
	w := &testhttp.TestResponseWriter{}
	NullSSO.HandleRedirect(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.StatusCode)
}
