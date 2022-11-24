package utils

import (
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argoerrors "github.com/argoproj/argo-workflows/v3/errors"

	apierr "k8s.io/apimachinery/pkg/api/errors"
)

func ArgoToStatusError(err argoerrors.ArgoError) error {
	switch err.Code() {
	case argoerrors.CodeUnauthorized:
		return status.Error(codes.Unauthenticated, err.Error())
	case argoerrors.CodeForbidden:
		return status.Error(codes.PermissionDenied, err.Error())
	case argoerrors.CodeNotFound:
		return status.Error(codes.NotFound, err.Error())
	case argoerrors.CodeBadRequest:
		return status.Error(codes.InvalidArgument, err.Error())
	case argoerrors.CodeNotImplemented:
		return status.Error(codes.Unimplemented, err.Error())
	case argoerrors.CodeTimeout:
		return status.Error(codes.DeadlineExceeded, err.Error())
	case argoerrors.CodeInternal:
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Error(codes.Unknown, err.Error())
	}
}

// Return a new status error and true else nil and false
// is meant to be as close to the opposite of grpc gateways status -> http error code converter
// exceptions made where one to one mappings were not available
func HTTPToStatusError(code int, msg string) (error, bool) {
	// handle success & information  codes in one go
	if code < 300 {
		return status.Error(codes.OK, msg), true
	}

	switch code {
	case http.StatusOK:
		return nil, true
	case http.StatusRequestTimeout:
		return status.Error(codes.Canceled, msg), true
	case http.StatusGatewayTimeout:
		return status.Error(codes.DeadlineExceeded, msg), true
	case http.StatusNotFound:
		return status.Error(codes.NotFound, msg), true
	case http.StatusConflict:
		return status.Error(codes.AlreadyExists, msg), true
	case http.StatusForbidden:
		return status.Error(codes.PermissionDenied, msg), true
	case http.StatusUnauthorized:
		return status.Error(codes.Unauthenticated, msg), true
	case http.StatusTooManyRequests:
		return status.Error(codes.ResourceExhausted, msg), true
	case http.StatusBadRequest:
		return status.Error(codes.FailedPrecondition, msg), true
	case http.StatusNotImplemented:
		return status.Error(codes.Unimplemented, msg), true
	case http.StatusInternalServerError:
		return status.Error(codes.Internal, msg), true
	case http.StatusServiceUnavailable:
		return status.Error(codes.Unavailable, msg), true
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
	argoerr, typeOkay := err.(argoerrors.ArgoError)
	if typeOkay {
		return ArgoToStatusError(argoerr)
	}
	// allow callers to call ToStatusError on the same processed error
	_, alreadyConverted := status.FromError(err)
	if alreadyConverted {
		return err
	}

	e := &apierr.StatusError{}
	if errors.As(err, &e) { // check if it's a Kubernetes API error
		// There is a http error code somewhere in the error stack
		statusCode := int(e.Status().Code)
		newErr, converted := HTTPToStatusError(statusCode, err.Error())
		if converted {
			return newErr
		}
	}
	return status.Error(code, err.Error())
}
