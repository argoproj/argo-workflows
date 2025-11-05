package sso

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
)

type HTTPClientConfig struct {
	InsecureSkipVerify bool
	RootCA             string
}

func (c HTTPClientConfig) String() string {
	rootCALen := len(c.RootCA)
	rootCAPreview := ""
	if rootCALen > 0 {
		if rootCALen > 50 {
			rootCAPreview = c.RootCA[:50] + "..."
		} else {
			rootCAPreview = c.RootCA
		}
	}

	return fmt.Sprintf("HTTPClientConfig{InsecureSkipVerify: %t, RootCA: %q (%d bytes)}",
		c.InsecureSkipVerify, rootCAPreview, rootCALen)
}

func createHTTPClient(config HTTPClientConfig) (*http.Client, error) {
	// Start with a copy of the default client
	httpClient := *http.DefaultClient

	// Clone the default transport and cast to *http.Transport
	defaultTransport := http.DefaultTransport.(*http.Transport)
	transport := defaultTransport.Clone()

	// Load system cert pool to respect env.SSL_CERT_DIR, env.SSL_CERT_FILE. macOS are not supported (https://pkg.go.dev/crypto/x509#SystemCertPool)
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system cert pool: %w", err)
	}

	// Set RootCAs if provided
	// Load root CA certificates from PEM string if defined
	if config.RootCA != "" {
		if ok := rootCAs.AppendCertsFromPEM([]byte(config.RootCA)); !ok {
			return nil, fmt.Errorf("failed to append certificates from PEM string")
		}
	}

	// Apply the custom TLS config to the cloned transport
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
		RootCAs:            rootCAs,
	}

	// Use the modified transport in our client copy
	httpClient.Transport = transport

	return &httpClient, nil
}
