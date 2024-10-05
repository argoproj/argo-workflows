package s3

import (
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

func TestIsTransientOSSErr(t *testing.T) {
	for _, errCode := range s3TransientErrorCodes {
		err := minio.ErrorResponse{Code: errCode}
		assert.True(t, isTransientS3Err(err))
	}

	err := minio.ErrorResponse{Code: "NoSuchBucket"}
	assert.False(t, isTransientS3Err(err))

	nonOSSErr := errors.New("UnseenError")
	assert.False(t, isTransientS3Err(nonOSSErr))

	requestErr := minio.ErrorResponse{Code: "RequestError"}
	assert.True(t, isTransientS3Err(requestErr))
}
