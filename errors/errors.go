package errors

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Externally visible error codes
const (
	CodeUnauthorized   = "ERR_UNAUTHORIZED"
	CodeBadRequest     = "ERR_BAD_REQUEST"
	CodeForbidden      = "ERR_FORBIDDEN"
	CodeNotFound       = "ERR_NOT_FOUND"
	CodeNotImplemented = "ERR_NOT_IMPLEMENTED"
	CodeTimeout        = "ERR_TIMEOUT"
	CodeInternal       = "ERR_INTERNAL"
)

// ArgoError is an error interface that additionally adds support for
// stack trace, error code, and a JSON representation of the error
type ArgoError interface {
	Error() string
	Code() string
	JSON() []byte
}

// argoerr is the internal implementation of an Argo error which wraps the error from pkg/errors
type argoerr struct {
	code    string
	message string
	err     error
}

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.
func New(code string, message string) error {
	err := errors.New(message)
	return argoerr{code, message, err}
}

// Errorf returns an error and formats according to a format specifier
func Errorf(code string, format string, args ...interface{}) error {
	return New(code, fmt.Sprintf(format, args...))
}

// InternalError is a convenience function to create a Internal error with a message
func InternalError(message string) error {
	return New(CodeInternal, message)
}

// InternalErrorf is a convenience function to format an Internal error
func InternalErrorf(format string, args ...interface{}) error {
	return Errorf(CodeInternal, format, args...)
}

// InternalWrapError annotates the error with the ERR_INTERNAL code and a stack trace, optional message
func InternalWrapError(err error, message ...string) error {
	if len(message) == 0 {
		return Wrap(err, CodeInternal, err.Error())
	}
	return Wrap(err, CodeInternal, message[0])
}

// InternalWrapErrorf annotates the error with the ERR_INTERNAL code and a stack trace, optional message
func InternalWrapErrorf(err error, format string, args ...interface{}) error {
	return Wrap(err, CodeInternal, fmt.Sprintf(format, args...))
}

// Wrap returns an error annotating err with a stack trace at the point Wrap is called,
// and a new supplied message. The previous original is preserved and accessible via Cause().
// If err is nil, Wrap returns nil.
func Wrap(err error, code string, message string) error {
	if err == nil {
		return nil
	}
	err = fmt.Errorf(message+": %w", err)
	return argoerr{code, message, err}
}

// Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following
// interface:
//
//     type causer interface {
//            Cause() error
//     }
//
// If the error does not implement Cause, the original error will
// be returned. If the error is nil, nil will be returned without further
// investigation.
func Cause(err error) error {
	if argoErr, ok := err.(argoerr); ok {
		return unwrapCauseArgoErr(argoErr.err)
	}
	return unwrapCause(err)
}

func unwrapCauseArgoErr(err error) error {
	innerErr := errors.Unwrap(err)
	for innerErr != nil {
		err = innerErr
		innerErr = errors.Unwrap(err)
	}
	return err
}

func unwrapCause(err error) error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

func (e argoerr) Error() string {
	return e.message
}

func (e argoerr) Code() string {
	return e.code
}

func (e argoerr) JSON() []byte {
	type errBean struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	eb := errBean{e.code, e.message}
	j, _ := json.Marshal(eb)
	return j
}

// IsCode is a helper to determine if the error is of a specific code
func IsCode(code string, err error) bool {
	if argoErr, ok := err.(argoerr); ok {
		return argoErr.code == code
	}
	return false
}
