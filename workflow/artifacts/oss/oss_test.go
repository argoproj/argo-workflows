package oss

import (
	"errors"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/stretchr/testify/require"
)

func TestIsTransientOSSErr(t *testing.T) {
	for _, errCode := range ossTransientErrorCodes {
		err := oss.ServiceError{Code: errCode}
		require.True(t, isTransientOSSErr(err))
	}

	err := oss.ServiceError{Code: "NonTransientErrorCode"}
	require.False(t, isTransientOSSErr(err))

	nonOSSErr := errors.New("Non-OSS error")
	require.False(t, isTransientOSSErr(nonOSSErr))

	require.False(t, isTransientOSSErr(nil))
}
