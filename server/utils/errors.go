package utils

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argoerrors "github.com/argoproj/argo-workflows/v4/errors"

	apierr "k8s.io/apimachinery/pkg/api/errors"
)

var errorMaps = map[int]codes.Code{
	http.StatusOK:                  codes.OK,
	http.StatusRequestTimeout:      codes.Canceled,
	http.StatusGatewayTimeout:      codes.DeadlineExceeded,
	http.StatusNotFound:            codes.NotFound,
	http.StatusConflict:            codes.AlreadyExists,
	http.StatusForbidden:           codes.PermissionDenied,
	http.StatusUnauthorized:        codes.Unauthenticated,
	http.StatusTooManyRequests:     codes.ResourceExhausted,
	http.StatusBadRequest:          codes.InvalidArgument,
	http.StatusNotImplemented:      codes.Unimplemented,
	http.StatusInternalServerError: codes.Internal,
	http.StatusServiceUnavailable:  codes.Unavailable,
}

// Return a new status error and true else nil and false
// is meant to be as close to the opposite of grpc gateways status -> http error code converter
// exceptions made where one to one mappings were not available
func httpToStatusError(code int, msg string) (error, bool) {
	// handle success & information  codes in one go
	if code < 300 {
		return status.Error(codes.OK, msg), true
	}

	statusCode, ok := errorMaps[code]
	if ok {
		return status.Error(statusCode, msg), true
	}

	// redirects don't make sense for servers
	// so that must imply an internal server error
	if code < 400 && code >= 300 {
		return status.Error(codes.Internal, msg), true
	}

	if code >= 500 {
		return status.Error(codes.Internal, msg), true
	}

	if code >= 400 {
		return status.Error(codes.InvalidArgument, msg), true
	}

	return nil, false
}

// Try to see if we can obtain a http
// error code from the k8s layer or the ArgoError layer
// if not we resort to a default value of `code`
// NOTE: errors passed of the type from grpc's status are not converted
// and returned as is. This is to keep user code as dumb as possible.
// The assumption here is that the error in the lowest layer of the error stack is the most relevant error.
func ToStatusError(err error, code codes.Code) error {
	if err == nil {
		return nil
	}
	// allow callers to call ToStatusError on the same processed error
	_, alreadyConverted := status.FromError(err)
	if alreadyConverted {
		return err
	}
	var argoerr argoerrors.ArgoError
	if errors.As(err, &argoerr) {
		newErr, converted := httpToStatusError(argoerr.HTTPCode(), err.Error())
		if converted {
			return newErr
		}
	}

	e := &apierr.StatusError{}
	if errors.As(err, &e) { // check if it's a Kubernetes API error
		// There is a http error code somewhere in the error stack
		statusCode := int(e.Status().Code)
		newErr, converted := httpToStatusError(statusCode, err.Error())
		if converted {
			return newErr
		}
	}
	return status.Error(code, err.Error())
}
