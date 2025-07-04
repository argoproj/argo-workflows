package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestIsTransientS3Err(t *testing.T) {
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))

	err := minio.ErrorResponse{Code: "InternalError"}
	assert.True(t, isTransientS3Err(ctx, err))

	err = minio.ErrorResponse{Code: "ServiceUnavailable"}
	assert.True(t, isTransientS3Err(ctx, err))

	nonTransientErr := minio.ErrorResponse{Code: "NoSuchKey"}
	assert.False(t, isTransientS3Err(ctx, nonTransientErr))

	nonTransientErr = minio.ErrorResponse{Code: "AccessDenied"}
	assert.False(t, isTransientS3Err(ctx, nonTransientErr))
}

func TestIsTransientOSSErr(t *testing.T) {
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))

	for _, errCode := range s3TransientErrorCodes {
		err := minio.ErrorResponse{Code: errCode}
		assert.True(t, isTransientS3Err(ctx, err))
	}

	err := minio.ErrorResponse{Code: "NoSuchBucket"}
	assert.False(t, isTransientS3Err(ctx, err))

	nonOSSErr := errors.New("UnseenError")
	assert.False(t, isTransientS3Err(ctx, nonOSSErr))

	requestErr := minio.ErrorResponse{Code: "RequestError"}
	assert.True(t, isTransientS3Err(ctx, requestErr))
}
