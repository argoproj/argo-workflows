package s3

import (
	argos3 "github.com/argoproj/pkg/s3"

	"github.com/argoproj/argo-workflows/v3/util/errors"
)

// s3TransientErrorCodes is a list of S3 error codes that are transient (retryable)
// Reference: https://github.com/minio/minio-go/blob/92fe50d14294782d96402deb861d442992038109/retry.go#L90-L102
var s3TransientErrorCodes = []string{
	"InternalError",
	"RequestTimeout",
	"Throttling",
	"ThrottlingException",
	"RequestLimitExceeded",
	"RequestThrottled",
	"InternalError",
	"SlowDown",
}

// isTransientS3Err checks if an minio.ErrorResponse error is transient (retryable)
func isTransientS3Err(err error) bool {
	if err == nil {
		return false
	}
	for _, transientErrCode := range s3TransientErrorCodes {
		if argos3.IsS3ErrCode(err, transientErrCode) {
			return true
		}
	}
	return errors.IsTransientErr(err)
}
