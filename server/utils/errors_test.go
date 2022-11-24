package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecursiveStatus(t *testing.T) {
	err := status.Error(codes.Canceled, "msg")
	newErr := ToStatusError(err, codes.Internal)
	statusErr := status.Convert(newErr)
	assert.Equal(t, codes.Canceled, statusErr.Code())
}
