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
