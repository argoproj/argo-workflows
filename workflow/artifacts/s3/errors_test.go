package s3

import (
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

func TestIsTransientOSSErr(t *testing.T) {
	for _, errCode := range s3TransientErrorCodes {
		err := minio.ErrorResponse{Code: errCode}
		require.True(t, isTransientS3Err(err))
	}

	err := minio.ErrorResponse{Code: "NoSuchBucket"}
	require.False(t, isTransientS3Err(err))

	nonOSSErr := errors.New("UnseenError")
	require.False(t, isTransientS3Err(nonOSSErr))

	requestErr := minio.ErrorResponse{Code: "RequestError"}
	require.True(t, isTransientS3Err(requestErr))
}
