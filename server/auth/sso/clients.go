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
	RootCA             string
	RootCAFile         string	
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

	// Create the HTTP client with the configured TLS settings.
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
		},
	}

	return httpClient, nil
}
