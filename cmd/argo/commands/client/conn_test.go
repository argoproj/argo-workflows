package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

func TestGetAuthString(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	t.Setenv("ARGO_TOKEN", "my-token")
	authString, err := GetAuthString(ctx)
	require.NoError(t, err)
	assert.Equal(t, "my-token", authString)
}

func TestNamespace(t *testing.T) {
	t.Setenv("ARGO_NAMESPACE", "my-ns")
	ctx := logging.TestContext(t.Context())
	assert.Equal(t, "my-ns", Namespace(ctx))
}

func TestCreateOfflineClient(t *testing.T) {
	t.Run("creating an offline client with no files should not fail", func(t *testing.T) {
		Offline = true
		OfflineFiles = []string{}
		ctx := logging.TestContext(t.Context())
		_, _, err := NewAPIClient(ctx)

		assert.NoError(t, err)
	})

	t.Run("creating an offline client with a non-existing file should fail", func(t *testing.T) {
		Offline = true
		OfflineFiles = []string{"non-existing-file"}
		ctx := logging.TestContext(t.Context())
		_, _, err := NewAPIClient(ctx)

		assert.Error(t, err)
	})
}

func TestNewAPIClientRequiresClientCertificateAndKey(t *testing.T) {
	originalOverrides := overrides
	originalArgoServerOpts := ArgoServerOpts
	t.Cleanup(func() {
		overrides = originalOverrides
		ArgoServerOpts = originalArgoServerOpts
	})

	tests := []struct {
		name       string
		clientCert string
		clientKey  string
	}{
		{
			name:       "certificate without key",
			clientCert: "client.crt",
		},
		{
			name:      "key without certificate",
			clientKey: "client.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			overrides.AuthInfo.ClientCertificate = tt.clientCert
			overrides.AuthInfo.ClientKey = tt.clientKey

			_, _, err := NewAPIClient(logging.TestContext(t.Context()))

			require.EqualError(t, err, "--client-certificate and --client-key must be provided together")
		})
	}
}

func TestNewAPIClientUsesExplicitCertificateAuthority(t *testing.T) {
	originalOverrides := overrides
	originalArgoServerOpts := ArgoServerOpts
	originalOffline := Offline
	originalOfflineFiles := OfflineFiles
	t.Cleanup(func() {
		overrides = originalOverrides
		ArgoServerOpts = originalArgoServerOpts
		Offline = originalOffline
		OfflineFiles = originalOfflineFiles
	})

	overrides.AuthInfo.ClientCertificate = ""
	overrides.AuthInfo.ClientKey = ""
	overrides.ClusterInfo.CertificateAuthority = "ca.crt"
	overrides.ClusterInfo.ProxyURL = ""
	Offline = true
	OfflineFiles = nil

	_, _, err := NewAPIClient(logging.TestContext(t.Context()))

	require.NoError(t, err)
	assert.Equal(t, "ca.crt", ArgoServerOpts.CACert)
}
