package tls

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("Create certificate with default options", func(t *testing.T) {
		certBytes, privKey, err := generate()
		require.NoError(t, err)
		assert.NotNil(t, privKey)
		cert, err := x509.ParseCertificate(certBytes)
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.Len(t, cert.DNSNames, 1)
		assert.Equal(t, "localhost", cert.DNSNames[0])
		assert.Empty(t, cert.IPAddresses)
		assert.LessOrEqual(t, int64(time.Since(cert.NotBefore)), int64(10*time.Second))
	})
}

func TestGeneratePEM(t *testing.T) {
	t.Run("Create PEM from certficate options", func(t *testing.T) {
		cert, key, err := generatePEM()
		require.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotNil(t, key)
	})

	t.Run("Create X509KeyPair", func(t *testing.T) {
		cert, err := GenerateX509KeyPair()
		require.NoError(t, err)
		assert.NotNil(t, cert)
	})
}
