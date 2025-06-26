package s3

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/util/errors"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

// s3TransientErrorCodes is a list of S3 error codes that are transient (retryable)
// Reference: https://github.com/minio/minio-go/blob/92fe50d14294782d96402deb861d442992038109/retry.go#L90-L102
var s3TransientErrorCodes = []string{
	"RequestError",
	"RequestTimeout",
	"Throttling",
	"ThrottlingException",
	"RequestLimitExceeded",
	"RequestThrottled",
	"InternalError",
	"SlowDown",
	"ServiceUnavailable",
}

// isTransientS3Err checks if an minio.ErrorResponse error is transient (retryable)
func isTransientS3Err(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	log := logging.GetLoggerFromContext(ctx)
	if log == nil {
		log = logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat())
	}
	for _, transientErrCode := range s3TransientErrorCodes {
		if IsS3ErrCode(err, transientErrCode) {
			log.Errorf(ctx, "Transient S3 error: %v", err)
			return true
		}
	}
	return errors.IsTransientErr(ctx, err)
}
