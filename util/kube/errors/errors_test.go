package errors

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	apierr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestIsRequestEntityTooLargeErr test
func TestIsRequestEntityTooLargeErr(t *testing.T) {
	assert.False(t, IsRequestEntityTooLargeErr(nil))

	err := &apierr.StatusError{ErrStatus: metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusRequestEntityTooLarge,
	}}
	assert.True(t, IsRequestEntityTooLargeErr(err))

	err = &apierr.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusInternalServerError,
		Message: "etcdserver: request is too large",
	}}
	assert.True(t, IsRequestEntityTooLargeErr(err))

	err = &apierr.StatusError{ErrStatus: metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusInternalServerError,
	}}
	assert.False(t, IsRequestEntityTooLargeErr(err))
}
