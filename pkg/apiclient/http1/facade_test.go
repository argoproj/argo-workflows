package http1

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tlsutil "github.com/argoproj/argo-workflows/v4/util/tls"
)

func TestFacade_do(t *testing.T) {
	f := Facade{baseURL: "http://my-url"}
	u, err := f.url("GET", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())

	u, err = f.url("DELETE", "/{namespace}/{name}", &metav1.ObjectMeta{Namespace: "my-ns", Labels: map[string]string{"foo": "1"}})
	require.NoError(t, err)
	assert.Equal(t, "http://my-url/my-ns/?labels.foo=1", u.String())
}

func TestNewFacade(t *testing.T) {
	clientCert, clientKey := writeClientKeyPair(t)
	caCert := clientCert

	t.Run("loads client certificate", func(t *testing.T) {
		f, err := NewFacade(FacadeConfig{
			InsecureSkipVerify: true,
			ClientCert:         clientCert,
			ClientKey:          clientKey,
			CACert:             caCert,
		})
		require.NoError(t, err)
		require.NotNil(t, f.tlsConfig)
		assert.True(t, f.tlsConfig.InsecureSkipVerify)
		assert.Equal(t, uint16(tls.VersionTLS12), f.tlsConfig.MinVersion)
		assert.Len(t, f.tlsConfig.Certificates, 1)
		assert.NotNil(t, f.tlsConfig.RootCAs)
	})

	t.Run("rejects incomplete client certificate pair", func(t *testing.T) {
		_, err := NewFacade(FacadeConfig{ClientCert: clientCert})
		require.ErrorContains(t, err, "requires both clientCert and clientKey")
	})

	t.Run("uses custom client without building default TLS config", func(t *testing.T) {
		customClient := &http.Client{}
		f, err := NewFacade(FacadeConfig{
			HTTPClient: customClient,
			ClientCert: "not-used.crt",
		})
		require.NoError(t, err)
		assert.Same(t, customClient, f.httpClient)
		assert.Nil(t, f.tlsConfig)
	})
}

func TestFacade_client(t *testing.T) {
	proxyErr := errors.New("proxy error")
	f := Facade{
		proxy: func(*http.Request) (*url.URL, error) {
			return nil, proxyErr
		},
		tlsConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
	require.NoError(t, err)

	client, err := f.client(req, false)
	require.ErrorIs(t, err, proxyErr)
	assert.Nil(t, client)

	f.proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	client, err = f.client(req, true)
	require.NoError(t, err)
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	assert.True(t, transport.DisableKeepAlives)
	assert.Same(t, f.tlsConfig, transport.TLSClientConfig)
}

func TestFacade_proxyFunc(t *testing.T) {
	proxyFunc := func(_ *http.Request) (*url.URL, error) {
		return nil, nil
	}
	tests := []struct {
		name  string
		proxy func(*http.Request) (*url.URL, error)
		want  func(*http.Request) (*url.URL, error)
	}{
		{
			name:  "use proxy settings from environment variables",
			proxy: nil,
			want:  http.ProxyFromEnvironment,
		},
		{
			name:  "use specific proxy",
			proxy: proxyFunc,
			want:  proxyFunc,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Facade{proxy: tt.want}
			got := f.proxyFunc()
			if reflect.ValueOf(got).Pointer() != reflect.ValueOf(tt.want).Pointer() {
				t.Errorf("Facade.proxyURL() = %p, want %p", got, tt.want)
			}
		})
	}
}

func writeClientKeyPair(t *testing.T) (string, string) {
	t.Helper()
	keyPair, err := tlsutil.GenerateX509KeyPair()
	require.NoError(t, err)
	privateKey, ok := keyPair.PrivateKey.(*ecdsa.PrivateKey)
	require.True(t, ok)
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "client.crt")
	keyPath := filepath.Join(tmpDir, "client.key")
	require.NoError(t, os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: keyPair.Certificate[0]}), 0o600))
	require.NoError(t, os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600))
	return certPath, keyPath
}
