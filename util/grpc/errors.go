package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	apierr "k8s.io/apimachinery/pkg/api/errors"
)

// translate a K8S errors into gRPC error - assume that we want to surface this - which we may not
func TranslateError(err error) error {
	switch {
	case err == nil:
		return nil
	case apierr.IsNotFound(err):
		return status.Error(codes.NotFound, err.Error())
	case apierr.IsAlreadyExists(err):
		return status.Error(codes.AlreadyExists, err.Error())
	case apierr.IsInvalid(err):
		return status.Error(codes.InvalidArgument, err.Error())
	case apierr.IsMethodNotSupported(err):
		return status.Error(codes.Unimplemented, err.Error())
	case apierr.IsServiceUnavailable(err):
		return status.Error(codes.Unavailable, err.Error())
	case apierr.IsBadRequest(err):
		return status.Error(codes.FailedPrecondition, err.Error())
	case apierr.IsUnauthorized(err):
		return status.Error(codes.Unauthenticated, err.Error())
	case apierr.IsForbidden(err):
		return status.Error(codes.PermissionDenied, err.Error())
	case apierr.IsTimeout(err):
		return status.Error(codes.DeadlineExceeded, err.Error())
	case apierr.IsInternalError(err):
		return status.Error(codes.Internal, err.Error())
	}
	return err
}
