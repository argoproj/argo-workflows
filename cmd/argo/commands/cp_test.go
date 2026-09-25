package commands

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/pkg/apiclient"
)

func TestNewArtifactHTTPClient(t *testing.T) {
	t.Run("configures TLS", func(t *testing.T) {
		client, err := newArtifactHTTPClient(apiclient.ArgoServerOpts{InsecureSkipVerify: true})
		require.NoError(t, err)
		transport, ok := client.Transport.(*http.Transport)
		require.True(t, ok)
		assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), transport.TLSClientConfig.MinVersion)
	})

	t.Run("rejects incomplete client certificate pair", func(t *testing.T) {
		client, err := newArtifactHTTPClient(apiclient.ArgoServerOpts{ClientCert: "client.crt"})
		require.ErrorContains(t, err, "requires both clientCert and clientKey")
		assert.Nil(t, client)
	})
}
