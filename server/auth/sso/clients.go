package sso

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

type HTTPClientConfig struct {
	InsecureSkipVerify bool
	RootCA             string
	RootCAFile         string
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

	return fmt.Sprintf("HTTPClientConfig{InsecureSkipVerify: %t, RootCA: %q (%d bytes), RootCAFile: %q}",
		c.InsecureSkipVerify, rootCAPreview, rootCALen, c.RootCAFile)
}

func createHTTPClient(config HTTPClientConfig) (*http.Client, error) {

	// Start with a copy of the default client
	httpClient := *http.DefaultClient

	// If no custom TLS configuration is needed, return the default client copy
	if !config.InsecureSkipVerify && config.RootCA == "" && config.RootCAFile == "" {
		return &httpClient, nil
	}

	// Clone the default transport and cast to *http.Transport
	defaultTransport := http.DefaultTransport.(*http.Transport)
	transport := defaultTransport.Clone()

	// Configure TLS settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	// Set RootCAs if provided
	// Load root CA certificates from both string and file if defined
	if config.RootCA != "" || config.RootCAFile != "" {

		rootCAs := x509.NewCertPool()

		// Add certificates from PEM string if provided
		if config.RootCA != "" {
			if ok := rootCAs.AppendCertsFromPEM([]byte(config.RootCA)); !ok {
				return nil, fmt.Errorf("failed to append certificates from PEM string")
			}
		}

		// Add certificates from file if provided
		if config.RootCAFile != "" {
			rootCAFile, err := os.ReadFile(config.RootCAFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read CA certificate file: %w", err)
			}

			if ok := rootCAs.AppendCertsFromPEM(rootCAFile); !ok {
				return nil, fmt.Errorf("failed to append CA certificate from file")
			}
		}

		tlsConfig.RootCAs = rootCAs
	}

	// Apply the custom TLS config to the cloned transport
	transport.TLSClientConfig = tlsConfig

	// Use the modified transport in our client copy
	httpClient.Transport = transport

	return &httpClient, nil
}
