package client

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
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

func TestCreateOfflineClient(t *testing.T) {
	t.Run("creating an offline client with no files should not fail", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		Offline = true
		OfflineFiles = []string{}
		NewAPIClient(context.TODO())

		assert.False(t, fatal, "should have exited")
	})

	t.Run("creating an offline client with a non-existing file should fail", func(t *testing.T) {
		defer func() { logrus.StandardLogger().ExitFunc = nil }()
		var fatal bool
		logrus.StandardLogger().ExitFunc = func(int) { fatal = true }

		Offline = true
		OfflineFiles = []string{"non-existing-file"}
		NewAPIClient(context.TODO())

		assert.True(t, fatal, "should have exited")
	})
}
