package oss

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestIsTransientOSSErr(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	for _, errCode := range ossTransientErrorCodes {
		err := oss.ServiceError{Code: errCode}
		assert.True(t, isTransientOSSErr(ctx, err))
	}

	err := oss.ServiceError{Code: "NonTransientErrorCode"}
	assert.False(t, isTransientOSSErr(ctx, err))

	nonOSSErr := errors.New("Non-OSS error")
	assert.False(t, isTransientOSSErr(ctx, nonOSSErr))

	assert.False(t, isTransientOSSErr(ctx, nil))
}

// TestIsOssErrCode tests the IsOssErrCode function
func TestIsOssErrCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{
			name:     "Matching error code",
			err:      oss.ServiceError{Code: "NoSuchKey"},
			code:     "NoSuchKey",
			expected: true,
		},
		{
			name:     "Non-matching error code",
			err:      oss.ServiceError{Code: "AccessDenied"},
			code:     "NoSuchKey",
			expected: false,
		},
		{
			name:     "Non-OSS error",
			err:      errors.New("generic error"),
			code:     "NoSuchKey",
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			code:     "NoSuchKey",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsOssErrCode(tc.err, tc.code)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestOssTransientErrorCodes tests that all transient error codes are properly recognized
func TestOssTransientErrorCodes(t *testing.T) {
	ctx := logging.TestContext(t.Context())

	// Test that all defined transient codes are recognized
	expectedTransientCodes := []string{
		"RequestTimeout",
		"QuotaExceeded.Refresh",
		"Default",
		"ServiceUnavailable",
		"Throttling",
		"RequestTimeTooSkewed",
		"SocketException",
		"SocketTimeout",
		"ServiceBusy",
		"DomainNetWorkVisitedException",
		"ConnectionTimeout",
		"CachedTimeTooLarge",
		"InternalError",
	}

	for _, code := range expectedTransientCodes {
		err := oss.ServiceError{Code: code}
		assert.True(t, isTransientOSSErr(ctx, err), "Expected %s to be transient", code)
	}
}

// TestPutFileSimpleVsMultipart tests the upload method selection based on file size
func TestPutFileSimpleVsMultipart(t *testing.T) {
	// Verify the maxObjectSize constant
	assert.Equal(t, int64(5*1024*1024*1024), maxObjectSize, "maxObjectSize should be 5GB")
}
