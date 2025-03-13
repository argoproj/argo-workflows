package sso

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"
)

type HttpClientConfig struct {
	ClientCert         string
	ClientKey          string
	InsecureSkipVerify bool
	CACert             string
}

func createHttpClient(config HttpClientConfig) (*http.Client, error) {
	var tlsConfig tls.Config
	// Only set certificates if both ClientCert and ClientKey are provided
	if config.ClientCert != "" && config.ClientKey != "" {
		cert, err := tls.LoadX509KeyPair(config.ClientCert, config.ClientKey)
		if err != nil {
			return nil, fmt.Errorf("unable to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	tlsConfig.InsecureSkipVerify = config.InsecureSkipVerify

	// Set RootCAs if provided
	if config.CACert != "" {
		// Load the CA certificate(s)
		rootCAs := x509.NewCertPool()
		caCert, err := os.ReadFile(config.CACert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		if ok := rootCAs.AppendCertsFromPEM(caCert); !ok {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
		tlsConfig.RootCAs = rootCAs
	}

	// Create the HTTP client with the configured TLS settings.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	return httpClient, nil
}
