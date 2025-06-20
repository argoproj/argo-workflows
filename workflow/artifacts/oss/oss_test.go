package oss

import (
	"context"
	"errors"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/assert"
)

func TestIsTransientOSSErr(t *testing.T) {
	for _, errCode := range ossTransientErrorCodes {
		err := oss.ServiceError{Code: errCode}
		assert.True(t, isTransientOSSErr(context.Background(), err))
	}

	err := oss.ServiceError{Code: "NonTransientErrorCode"}
	assert.False(t, isTransientOSSErr(context.Background(), err))

	nonOSSErr := errors.New("Non-OSS error")
	assert.False(t, isTransientOSSErr(context.Background(), nonOSSErr))

	assert.False(t, isTransientOSSErr(context.Background(), nil))
}
