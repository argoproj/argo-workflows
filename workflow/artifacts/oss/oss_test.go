package oss

import (
	"context"
	"errors"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"

	"github.com/argoproj/argo-workflows/v3/util/logging"
)

func TestIsTransientOSSErr(t *testing.T) {
	ctx := context.Background()
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))
	ctx = logging.WithLogger(ctx, logging.NewSlogLogger(logging.GetGlobalLevel(), logging.GetGlobalFormat()))

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
