package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/errors"
)

// TestErrorf tests the initializer of error package
func TestErrorf(t *testing.T) {
	err := errors.Errorf(errors.CodeInternal, "test internal")
	assert.Equal(t, "test internal", err.Error())
}

// TestWrap ensures we can wrap an error and use Cause() to retrieve the original error
func TestWrap(t *testing.T) {
	err := fmt.Errorf("original error message")
	argoErr := errors.Wrap(err, "WRAPPED", "wrapped message")
	assert.Equal(t, "wrapped message", argoErr.Error())
	orig := errors.Cause(argoErr)
	assert.Equal(t, err.Error(), orig.Error())
}

// TestInternalError verifies
func TestInternalError(t *testing.T) {
	err := errors.InternalError("test internal")
	assert.Equal(t, "test internal", err.Error())

	// Test wrapping errors
	err = fmt.Errorf("random error")
	intWrap := errors.InternalWrapError(err)
	assert.Equal(t, "random error", intWrap.Error())
	intWrap = errors.InternalWrapError(err, "different message")
	assert.Equal(t, "different message", intWrap.Error())
	intWrap = errors.InternalWrapErrorf(err, "hello %s", "world")
	assert.Equal(t, "hello world", intWrap.Error())
}
