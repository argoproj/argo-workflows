package s3

import (
	"errors"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestIsTransientS3Err(t *testing.T) {
	ctx := logging.TestContext(t.Context())

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
	ctx := logging.TestContext(t.Context())

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

func TestIsTransientS3Err_BareHTTPStatus(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// minio-go falls back to resp.Status as Code when the error body is not
	// parsable S3 XML (e.g. a load balancer returned a plain 5xx response).
	bare503 := minio.ErrorResponse{Code: "503 Service Unavailable", StatusCode: 503}
	assert.True(t, isTransientS3Err(ctx, bare503))

	bare500 := minio.ErrorResponse{Code: "500 Internal Server Error", StatusCode: 500}
	assert.True(t, isTransientS3Err(ctx, bare500))

	bare404 := minio.ErrorResponse{Code: "404 Not Found", StatusCode: 404}
	assert.False(t, isTransientS3Err(ctx, bare404))
}
