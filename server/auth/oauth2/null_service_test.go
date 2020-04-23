package oauth2

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func Test_nullService_Authorize(t *testing.T) {
	user, err := NullService.Authorize(context.Background(), "")
	if assert.Error(t, err) {
		assert.Equal(t, wfv1.NullUser, user)
	}
}

func Test_nullService_HandleCallback(t *testing.T) {
	w := &testhttp.TestResponseWriter{}
	NullService.HandleCallback(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.StatusCode)
}

func Test_nullService_HandleRedirect(t *testing.T) {
	w := &testhttp.TestResponseWriter{}
	NullService.HandleRedirect(w, &http.Request{})
	assert.Equal(t, http.StatusNotImplemented, w.StatusCode)
}

func Test_nullService_IsSSO(t *testing.T) {
	assert.False(t, NullService.IsSSO(""))
}
