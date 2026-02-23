package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"k8s.io/client-go/kubernetes"

	"github.com/argoproj/argo-workflows/v3/util"
)

const (
	// The key of the tls.crt within the Kubernetes secret
	tlsCrtSecretKey = "tls.crt"

	// The key of the tls.key within the Kubernetes secret
	tlsKeySecretKey = "tls.key"
)

func pemBlockForKey(priv any) *pem.Block {
	switch k := priv.(type) {
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			log.Print(err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

func generate() ([]byte, crypto.PrivateKey, error) {
	hosts := []string{"localhost"}

	var err error
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ArgoProj"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}
	return certBytes, privateKey, nil
}

// generatePEM generates a new certificate and key and returns it as PEM encoded bytes
func generatePEM() ([]byte, []byte, error) {
	certBytes, privateKey, err := generate()
	if err != nil {
		return nil, nil, err
	}
	certpem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keypem := pem.EncodeToMemory(pemBlockForKey(privateKey))
	return certpem, keypem, nil
}

// GenerateX509KeyPair generates a X509 key pair
func GenerateX509KeyPair() (*tls.Certificate, error) {
	certpem, keypem, err := generatePEM()
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(certpem, keypem)
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

func GenerateX509KeyPairTLSConfig(tlsMinVersion uint16) (*tls.Config, error) {
	cer, err := GenerateX509KeyPair()
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{*cer},
		MinVersion:         tlsMinVersion,
		InsecureSkipVerify: true,
		NextProtos:         []string{"h2"},
	}, nil
}

func GetServerTLSConfigFromSecret(ctx context.Context, kubectlConfig kubernetes.Interface, tlsKubernetesSecretName string, tlsMinVersion uint16, namespace string) (*tls.Config, error) {
	certpem, err := util.GetSecrets(ctx, kubectlConfig, namespace, tlsKubernetesSecretName, tlsCrtSecretKey)
	if err != nil {
		return nil, err
	}

	keypem, err := util.GetSecrets(ctx, kubectlConfig, namespace, tlsKubernetesSecretName, tlsKeySecretKey)
	if err != nil {
		return nil, err
	}

	cert, err := tls.X509KeyPair(certpem, keypem)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tlsMinVersion,
		NextProtos:   []string{"h2"},
	}, nil
}
