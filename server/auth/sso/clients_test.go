package sso

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generateTestCert creates a valid self-signed certificate for testing
func generateTestCert() (string, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test-ca",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return "", err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	return string(certPEM), nil
}

func TestHTTPClientConfig_String(t *testing.T) {
	tests := []struct {
		name     string
		config   HTTPClientConfig
		expected string
	}{
		{
			name: "empty config",
			config: HTTPClientConfig{
				InsecureSkipVerify: false,
				RootCA:             "",
				RootCAFile:         "",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: false, RootCA: "" (0 bytes), RootCAFile: ""}`,
		},
		{
			name: "insecure skip verify true",
			config: HTTPClientConfig{
				InsecureSkipVerify: true,
				RootCA:             "",
				RootCAFile:         "",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: true, RootCA: "" (0 bytes), RootCAFile: ""}`,
		},
		{
			name: "short root CA",
			config: HTTPClientConfig{
				InsecureSkipVerify: false,
				RootCA:             "short-ca-content",
				RootCAFile:         "",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: false, RootCA: "short-ca-content" (16 bytes), RootCAFile: ""}`,
		},
		{
			name: "long root CA gets truncated",
			config: HTTPClientConfig{
				InsecureSkipVerify: false,
				RootCA:             strings.Repeat("a", 100),
				RootCAFile:         "",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: false, RootCA: "` + strings.Repeat("a", 50) + `..." (100 bytes), RootCAFile: ""}`,
		},
		{
			name: "with root CA file",
			config: HTTPClientConfig{
				InsecureSkipVerify: false,
				RootCA:             "",
				RootCAFile:         "/path/to/ca.pem",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: false, RootCA: "" (0 bytes), RootCAFile: "/path/to/ca.pem"}`,
		},
		{
			name: "all fields set",
			config: HTTPClientConfig{
				InsecureSkipVerify: true,
				RootCA:             "test-ca",
				RootCAFile:         "/path/to/ca.pem",
			},
			expected: `HTTPClientConfig{InsecureSkipVerify: true, RootCA: "test-ca" (7 bytes), RootCAFile: "/path/to/ca.pem"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.String()
			if result != tt.expected {
				t.Errorf("HTTPClientConfig.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestCreateHTTPClient_DefaultConfig(t *testing.T) {
	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             "",
		RootCAFile:         "",
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should return a copy of the default client with default transport
	transport, ok := client.Transport.(*http.Transport)
	if ok && transport.TLSClientConfig != nil {
		t.Error("Expected default transport for default config, but got custom TLS config")
	}
}

func TestCreateHTTPClient_InsecureSkipVerify(t *testing.T) {
	config := HTTPClientConfig{
		InsecureSkipVerify: true,
		RootCA:             "",
		RootCAFile:         "",
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should have custom transport with InsecureSkipVerify = true
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}
}

func TestCreateHTTPClient_WithRootCAString(t *testing.T) {
	testCertPEM, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             testCertPEM,
		RootCAFile:         "",
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should have custom transport with RootCAs set
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}
}

func TestCreateHTTPClient_WithRootCAFile(t *testing.T) {
	testCertPEM, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Create a temporary file with test certificate
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")
	
	err = os.WriteFile(caFile, []byte(testCertPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CA file: %v", err)
	}

	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             "",
		RootCAFile:         caFile,
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should have custom transport with RootCAs set
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}
}

func TestCreateHTTPClient_WithBothRootCAStringAndFile(t *testing.T) {
	testCertPEM, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Create a temporary file with test certificate
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")
	
	err = os.WriteFile(caFile, []byte(testCertPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CA file: %v", err)
	}

	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             testCertPEM,
		RootCAFile:         caFile,
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should have custom transport with RootCAs set
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}
}

func TestCreateHTTPClient_InvalidRootCAString(t *testing.T) {
	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             "invalid-pem-content",
		RootCAFile:         "",
	}

	client, err := createHTTPClient(config)
	if err == nil {
		t.Fatal("Expected error for invalid PEM content, got nil")
	}

	if client != nil {
		t.Error("Expected nil client for invalid PEM content")
	}

	expectedError := "failed to append certificates from PEM string"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %q", expectedError, err.Error())
	}
}

func TestCreateHTTPClient_NonExistentRootCAFile(t *testing.T) {
	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             "",
		RootCAFile:         "/nonexistent/path/ca.pem",
	}

	client, err := createHTTPClient(config)
	if err == nil {
		t.Fatal("Expected error for nonexistent file, got nil")
	}

	if client != nil {
		t.Error("Expected nil client for nonexistent file")
	}

	expectedError := "failed to read CA certificate file"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %q", expectedError, err.Error())
	}
}

func TestCreateHTTPClient_InvalidRootCAFile(t *testing.T) {
	// Create a temporary file with invalid certificate content
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "invalid-ca.pem")
	
	err := os.WriteFile(caFile, []byte("invalid-pem-content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CA file: %v", err)
	}

	config := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             "",
		RootCAFile:         caFile,
	}

	client, err := createHTTPClient(config)
	if err == nil {
		t.Fatal("Expected error for invalid PEM file content, got nil")
	}

	if client != nil {
		t.Error("Expected nil client for invalid PEM file content")
	}

	expectedError := "failed to append CA certificate from file"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("Expected error to contain %q, got %q", expectedError, err.Error())
	}
}

func TestCreateHTTPClient_AllOptionsEnabled(t *testing.T) {
	testCertPEM, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	// Create a temporary file with test certificate
	tmpDir := t.TempDir()
	caFile := filepath.Join(tmpDir, "ca.pem")
	
	err = os.WriteFile(caFile, []byte(testCertPEM), 0644)
	if err != nil {
		t.Fatalf("Failed to create test CA file: %v", err)
	}

	config := HTTPClientConfig{
		InsecureSkipVerify: true,
		RootCA:             testCertPEM,
		RootCAFile:         caFile,
	}

	client, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	if client == nil {
		t.Fatal("createHTTPClient() returned nil client")
	}

	// Should have custom transport with both InsecureSkipVerify and RootCAs set
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Expected *http.Transport, got different type")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("Expected TLS config to be set")
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected InsecureSkipVerify to be true")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Error("Expected RootCAs to be set")
	}
}

func TestCreateHTTPClient_TransportCloning(t *testing.T) {
	config := HTTPClientConfig{
		InsecureSkipVerify: true,
		RootCA:             "",
		RootCAFile:         "",
	}

	client1, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	client2, err := createHTTPClient(config)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	// Ensure that each client gets its own transport instance
	if client1.Transport == client2.Transport {
		t.Error("Expected different transport instances for different clients")
	}

	// Ensure that neither client uses the default transport
	if client1.Transport == http.DefaultTransport {
		t.Error("Expected client1 to have custom transport, not default transport")
	}

	if client2.Transport == http.DefaultTransport {
		t.Error("Expected client2 to have custom transport, not default transport")
	}
}

func TestCreateHTTPClient_TLSConfigIsolation(t *testing.T) {
	config1 := HTTPClientConfig{
		InsecureSkipVerify: true,
		RootCA:             "",
		RootCAFile:         "",
	}

	testCertPEM, err := generateTestCert()
	if err != nil {
		t.Fatalf("Failed to generate test certificate: %v", err)
	}

	config2 := HTTPClientConfig{
		InsecureSkipVerify: false,
		RootCA:             testCertPEM,
		RootCAFile:         "",
	}

	client1, err := createHTTPClient(config1)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	client2, err := createHTTPClient(config2)
	if err != nil {
		t.Fatalf("createHTTPClient() error = %v, want nil", err)
	}

	transport1 := client1.Transport.(*http.Transport)
	transport2 := client2.Transport.(*http.Transport)

	// Ensure TLS configs are different
	if transport1.TLSClientConfig == transport2.TLSClientConfig {
		t.Error("Expected different TLS config instances for different clients")
	}

	// Verify specific settings
	if !transport1.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected client1 to have InsecureSkipVerify = true")
	}

	if transport2.TLSClientConfig.InsecureSkipVerify {
		t.Error("Expected client2 to have InsecureSkipVerify = false")
	}

	if transport1.TLSClientConfig.RootCAs != nil {
		t.Error("Expected client1 to have no custom RootCAs")
	}

	if transport2.TLSClientConfig.RootCAs == nil {
		t.Error("Expected client2 to have custom RootCAs")
	}
}
