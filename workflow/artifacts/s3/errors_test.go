package s3

import (
	"context"
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

func TestIsTransientS3Err(t *testing.T) {
	err := minio.ErrorResponse{Code: "InternalError"}
	assert.True(t, isTransientS3Err(context.Background(), err))

	err = minio.ErrorResponse{Code: "ServiceUnavailable"}
	assert.True(t, isTransientS3Err(context.Background(), err))

	nonTransientErr := minio.ErrorResponse{Code: "NoSuchKey"}
	assert.False(t, isTransientS3Err(context.Background(), nonTransientErr))

	nonTransientErr = minio.ErrorResponse{Code: "AccessDenied"}
	assert.False(t, isTransientS3Err(context.Background(), nonTransientErr))
}

func TestIsTransientOSSErr(t *testing.T) {
	for _, errCode := range s3TransientErrorCodes {
		err := minio.ErrorResponse{Code: errCode}
		assert.True(t, isTransientS3Err(context.Background(), err))
	}

	err := minio.ErrorResponse{Code: "NoSuchBucket"}
	assert.False(t, isTransientS3Err(context.Background(), err))

	nonOSSErr := errors.New("UnseenError")
	assert.False(t, isTransientS3Err(context.Background(), nonOSSErr))

	requestErr := minio.ErrorResponse{Code: "RequestError"}
	assert.True(t, isTransientS3Err(context.Background(), requestErr))
}
