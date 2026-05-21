package s3

import (
	"context"
	stderrors "errors"

	"github.com/minio/minio-go/v7"

	"github.com/argoproj/argo-workflows/v4/util/errors"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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
	log := logging.RequireLoggerFromContext(ctx)
	for _, transientErrCode := range s3TransientErrorCodes {
		if IsS3ErrCode(err, transientErrCode) {
			log.WithError(err).Error(ctx, "Transient S3 error")
			return true
		}
	}
	// When the response body is not a parsable S3 XML document (e.g. a proxy
	// or load balancer returned a bare 5xx response), minio-go sets Code to
	// the raw HTTP status string ("503 Service Unavailable"), which does not
	// match any entry in s3TransientErrorCodes. Fall back to StatusCode so
	// 5xx responses are still treated as transient per S3 retry semantics.
	var minioErr minio.ErrorResponse
	if stderrors.As(err, &minioErr) && minioErr.StatusCode >= 500 && minioErr.StatusCode < 600 {
		log.WithError(err).Error(ctx, "Transient S3 error")
		return true
	}
	return errors.IsTransientErr(ctx, err)
}
