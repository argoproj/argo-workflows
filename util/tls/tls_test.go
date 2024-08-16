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
	t.Run("Create PEM from certificate options", func(t *testing.T) {
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

func TestGetTLSConfig(t *testing.T) {
	tests := []struct {
		name               string
		clientCert         string
		clientKey          string
		insecureSkipVerify bool
		wantErr            bool
	}{
		{
			name:               "Valid certificate and key",
			clientCert:         "testdata/valid_tls.crt",
			clientKey:          "testdata/valid_tls.key",
			insecureSkipVerify: false,
			wantErr:            false,
		},
		{
			name:               "Empty certificate and key",
			clientCert:         "",
			clientKey:          "",
			insecureSkipVerify: true,
			wantErr:            false,
		},
		{
			name:       "Invalid certificate and key",
			clientCert: "testdata/empty_tls.crt",
			clientKey:  "testdata/empty_tls.key",
			wantErr:    true,
		},
		{
			name:       "Missing key file",
			clientCert: "testdata/valid_tls.crt",
			clientKey:  "testdata/nonexistent.key",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := GetTLSConfig(tt.clientCert, tt.clientKey, tt.insecureSkipVerify)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, config)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, config)
			assert.Equal(t, tt.insecureSkipVerify, config.InsecureSkipVerify)

			if tt.clientCert != "" && tt.clientKey != "" {
				assert.Len(t, config.Certificates, 1)
			} else {
				assert.Empty(t, config.Certificates)
			}
		})
	}
}
