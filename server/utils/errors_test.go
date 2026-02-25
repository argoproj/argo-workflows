package utils

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"
)

type testArgoError struct {
	code string
}

func (t testArgoError) Error() string {
	return "Test Error"
}

func (t testArgoError) Code() string {
	return t.code
}

func (t testArgoError) HTTPCode() int {
	switch t.Code() {
	case argoerrors.CodeUnauthorized:
		return http.StatusUnauthorized
	case argoerrors.CodeForbidden:
		return http.StatusForbidden
	case argoerrors.CodeNotFound:
		return http.StatusNotFound
	case argoerrors.CodeBadRequest:
		return http.StatusBadRequest
	case argoerrors.CodeNotImplemented:
		return http.StatusNotImplemented
	case argoerrors.CodeTimeout, argoerrors.CodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

func (t testArgoError) JSON() []byte {
	type errBean struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	eb := errBean{t.code, "Test Error Message"}
	j, _ := json.Marshal(eb)
	return j
}

func TestRecursiveStatus(t *testing.T) {
	err := status.Error(codes.Canceled, "msg")
	newErr := ToStatusError(err, codes.Internal)
	statusErr := status.Convert(newErr)
	assert.Equal(t, codes.Canceled, statusErr.Code())
}

func TestNilStatus(t *testing.T) {
	newErr := ToStatusError(nil, codes.InvalidArgument)
	require.NoError(t, newErr)
}

func TestArgoError(t *testing.T) {

	t.Run("CodeBadRequest", func(t *testing.T) {
		argoErr := testArgoError{argoerrors.CodeBadRequest}
		newErr := ToStatusError(argoErr, codes.Internal)
		stat := status.Convert(newErr)
		assert.Equal(t, codes.InvalidArgument, stat.Code())
	})

	t.Run("CodePermissionDenied", func(t *testing.T) {
		argoErr := testArgoError{argoerrors.CodeForbidden}
		newErr := ToStatusError(argoErr, codes.Internal)
		stat := status.Convert(newErr)
		assert.Equal(t, codes.PermissionDenied, stat.Code())
	})

	t.Run("CodeUnknown", func(t *testing.T) {
		argoErr := testArgoError{"UNKNOWN_ERR"}
		newErr := ToStatusError(argoErr, codes.Internal)
		stat := status.Convert(newErr)
		assert.Equal(t, codes.Internal, stat.Code())
	})

}

func TestHTTPToStatusError(t *testing.T) {
	assert := assert.New(t)

	t.Run("StatusOk", func(t *testing.T) {
		code := http.StatusAccepted
		err, ok := httpToStatusError(code, "msg")
		assert.True(ok)
		stat := status.Convert(err)
		assert.Equal(codes.OK, stat.Code())
	})

	t.Run("StatusOnRedirect", func(t *testing.T) {
		code := http.StatusPermanentRedirect
		err, ok := httpToStatusError(code, "msg")
		assert.True(ok)
		stat := status.Convert(err)
		assert.Equal(codes.Internal, stat.Code())
	})
	// Test 400 level errors not accounted for in map
	t.Run("StatusTeapot", func(t *testing.T) {
		code := http.StatusTeapot
		err, ok := httpToStatusError(code, "msg")
		assert.True(ok)
		stat := status.Convert(err)
		assert.Equal(codes.InvalidArgument, stat.Code())
	})

	// Test 500 level errors not accounted for in map
	t.Run("StatusInternal", func(t *testing.T) {
		code := http.StatusVariantAlsoNegotiates
		err, ok := httpToStatusError(code, "msg")
		assert.True(ok)
		stat := status.Convert(err)
		assert.Equal(codes.Internal, stat.Code())
	})
}
