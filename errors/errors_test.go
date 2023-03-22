package errors_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/errors"
)

// TestErrorf tests the initializer of error package
func TestErrorf(t *testing.T) {
	err := errors.Errorf(errors.CodeInternal, "test internal")
	assert.Equal(t, err.Error(), "test internal")
}

// TestWrap ensures we can wrap an error and use Cause() to retrieve the original error
func TestWrap(t *testing.T) {
	err := fmt.Errorf("original error message")
	argoErr := errors.Wrap(err, "WRAPPED", "wrapped message")
	assert.Equal(t, "wrapped message", argoErr.Error())
	orig := errors.Cause(argoErr)
	assert.Equal(t, err.Error(), orig.Error())
}
