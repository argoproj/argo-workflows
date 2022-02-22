package client

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAuthString(t *testing.T) {
	_ = os.Setenv("ARGO_TOKEN", "my-token")
	defer func() { _ = os.Unsetenv("ARGO_TOKEN") }()
	assert.Equal(t, "my-token", GetAuthString())
}

func TestNamespace(t *testing.T) {
	_ = os.Setenv("ARGO_NAMESPACE", "my-ns")
	defer func() { _ = os.Unsetenv("ARGO_NAMESPACE") }()
	assert.Equal(t, "my-ns", Namespace())
}
