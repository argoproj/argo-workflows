package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v3/errors"
)

// TestErrorf tests the initializer of error package
func TestErrorf(t *testing.T) {
	err := errors.Errorf(errors.CodeInternal, "test internal")
	require.Equal(t, "test internal", err.Error())
}

// TestWrap ensures we can wrap an error and use Cause() to retrieve the original error
func TestWrap(t *testing.T) {
	err := fmt.Errorf("original error message")
	argoErr := errors.Wrap(err, "WRAPPED", "wrapped message")
	require.Equal(t, "wrapped message", argoErr.Error())
	orig := errors.Cause(argoErr)
	require.Equal(t, err.Error(), orig.Error())
}

// TestInternalError verifies
func TestInternalError(t *testing.T) {
	err := errors.InternalError("test internal")
	require.Equal(t, "test internal", err.Error())

	// Test wrapping errors
	err = fmt.Errorf("random error")
	intWrap := errors.InternalWrapError(err)
	require.Equal(t, "random error", intWrap.Error())
	intWrap = errors.InternalWrapError(err, "different message")
	assert.Equal(t, "different message", intWrap.Error())
	intWrap = errors.InternalWrapErrorf(err, "hello %s", "world")
	require.Equal(t, "hello world", intWrap.Error())
}
