package errors_test

import (
	"fmt"
	"testing"

	"github.com/argoproj/argo/errors"
	pkgerr "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

// stackTracer is interface for error types that have a stack trace
type stackTracer interface {
	StackTrace() pkgerr.StackTrace
}

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

// TestInternalError verifies
func TestInternalError(t *testing.T) {
	err := errors.InternalError("test internal")
	assert.Equal(t, "test internal", err.Error())

	// Test wrapping errors
	err = fmt.Errorf("random error")
	intWrap := errors.InternalWrapError(err)
	_ = intWrap.(stackTracer)
	assert.Equal(t, "random error", intWrap.Error())
	intWrap = errors.InternalWrapError(err, "different message")
	_ = intWrap.(stackTracer)
	assert.Equal(t, "different message", intWrap.Error())
	intWrap = errors.InternalWrapErrorf(err, "hello %s", "world")
	_ = intWrap.(stackTracer)
	assert.Equal(t, "hello world", intWrap.Error())
}

func TestStackTrace(t *testing.T) {
	err := errors.New("MYCODE", "my message")
	assert.Contains(t, fmt.Sprintf("%+v", err), "errors_test.go")
}
