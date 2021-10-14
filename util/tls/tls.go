package tls

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/util"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

const (
	DefaultRSABits = 2048
	// The default TLS cipher suites to provide to clients - see https://cipherlist.eu for updates
	// Note that for TLS v1.3, cipher suites are not configurable and will be chosen automatically.
	DefaultTLSCipherSuite = "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:TLS_RSA_WITH_AES_256_GCM_SHA384"
	// The default minimum TLS version to provide to clients
	DefaultTLSMinVersion = "1.2"
	// The default maximum TLS version to provide to clients
	DefaultTLSMaxVersion = "1.3"
	// The key of the tls.crt within the Kubernetes secret
	tlsCrtSecretKey = "tls.crt"
	// The key of the tls.key within the Kubernetes secret
	tlsKeySecretKey = "tls.key"
)

var (
	tlsVersionByString = map[string]uint16{
		"1.0": tls.VersionTLS10,
		"1.1": tls.VersionTLS11,
		"1.2": tls.VersionTLS12,
		"1.3": tls.VersionTLS13,
	}
)

type CertOptions struct {
	// Hostnames and IPs to generate a certificate for
	Hosts []string
	// Name of organization in certificate
	Organization string
	// Creation date
	ValidFrom time.Time
	// Duration that certificate is valid for
	ValidFor time.Duration
	// whether this cert should be its own Certificate Authority
	IsCA bool
	// Size of RSA key to generate. Ignored if --ecdsa-curve is set
	RSABits int
	// ECDSA curve to use to generate a key. Valid values are P224, P256 (recommended), P384, P521
	ECDSACurve string
}

type ConfigCustomizer = func(*tls.Config)

// BestEffortSystemCertPool returns system cert pool as best effort, otherwise an empty cert pool
func BestEffortSystemCertPool() *x509.CertPool {
	rootCAs, _ := x509.SystemCertPool()
	if rootCAs == nil {
		return x509.NewCertPool()
	}
	return rootCAs
}

func getTLSVersionByString(version string) (uint16, error) {
	if version == "" {
		return 0, nil
	}
	if res, ok := tlsVersionByString[version]; ok {
		return res, nil
	}
	return 0, fmt.Errorf("%s is not valid TLS version", version)
}

// Parse colon separated string representation of TLS cipher suites into array of values usable by crypto/tls
func getTLSCipherSuitesByString(cipherSuites string) ([]uint16, error) {
	suiteMap := make(map[string]uint16)
	for _, s := range tls.CipherSuites() {
		suiteMap[s.Name] = s.ID
	}
	allowedSuites := make([]uint16, 0)
	for _, s := range strings.Split(cipherSuites, ":") {
		id, ok := suiteMap[strings.TrimSpace(s)]
		if ok {
			allowedSuites = append(allowedSuites, id)
		} else {
			return nil, fmt.Errorf("invalid cipher suite specified: %s", s)
		}
	}
	return allowedSuites, nil
}

// Return array of strings representing TLS versions
func tlsVersionsToStr(versions []uint16) []string {
	ret := make([]string, 0)
	for _, v := range versions {
		switch v {
		case tls.VersionTLS10:
			ret = append(ret, "1.0")
		case tls.VersionTLS11:
			ret = append(ret, "1.1")
		case tls.VersionTLS12:
			ret = append(ret, "1.2")
		case tls.VersionTLS13:
			ret = append(ret, "1.3")
		default:
			ret = append(ret, "unknown")
		}
	}
	return ret
}

func getTLSConfigCustomizer(minVersionStr, maxVersionStr, tlsCiphersStr string) (ConfigCustomizer, error) {
	minVersion, err := getTLSVersionByString(minVersionStr)
	if err != nil {
		return nil, err
	}
	maxVersion, err := getTLSVersionByString(maxVersionStr)
	if err != nil {
		return nil, err
	}
	if minVersion > maxVersion {
		return nil, fmt.Errorf("Minimum TLS version %s must not be higher than maximum TLS version %s", minVersionStr, maxVersionStr)
	}

	// Cipher suites for TLSv1.3 are not configurable
	if minVersion == tls.VersionTLS13 {
		if tlsCiphersStr != DefaultTLSCipherSuite {
			log.Warnf("TLSv1.3 cipher suites are not configurable, ignoring value of --tlsciphers")
		}
		tlsCiphersStr = ""
	}

	if tlsCiphersStr == "list" {
		fmt.Printf("Supported TLS ciphers:\n")
		for _, s := range tls.CipherSuites() {
			fmt.Printf("* %s (TLS versions: %s)\n", tls.CipherSuiteName(s.ID), strings.Join(tlsVersionsToStr(s.SupportedVersions), ", "))
		}
		os.Exit(0)
	}

	var cipherSuites []uint16
	if tlsCiphersStr != "" {
		cipherSuites, err = getTLSCipherSuitesByString(tlsCiphersStr)
		if err != nil {
			return nil, err
		}
	} else {
		cipherSuites = make([]uint16, 0)
	}

	return func(config *tls.Config) {
		config.MinVersion = minVersion
		config.MaxVersion = maxVersion
		config.CipherSuites = cipherSuites
	}, nil

}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}
func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			log.Fatal(err)
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
		return nil, nil, fmt.Errorf("failed to generate private key: %s", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %s", err)
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
		return nil, nil, fmt.Errorf("failed to create certificate: %s", err)
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
		Certificates: []tls.Certificate{*cer},
		MinVersion:   uint16(tlsMinVersion),
	}, nil
}

// EncodeX509KeyPair encodes a TLS Certificate into its pem encoded format for storage
func EncodeX509KeyPair(cert tls.Certificate) ([]byte, []byte) {

	certpem := []byte{}
	for _, certtmp := range cert.Certificate {
		certpem = append(certpem, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certtmp})...)
	}
	keypem := pem.EncodeToMemory(pemBlockForKey(cert.PrivateKey))
	return certpem, keypem
}

// EncodeX509KeyPairString encodes a TLS Certificate into its pem encoded string format
func EncodeX509KeyPairString(cert tls.Certificate) (string, string) {
	certpem, keypem := EncodeX509KeyPair(cert)
	return string(certpem), string(keypem)
}

// LoadX509CertPool loads PEM data from a list of files, adds them to a CertPool
// and returns the resulting CertPool
func LoadX509CertPool(paths ...string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, path := range paths {
		log.Infof("Loading CA information from %s and appending it to cert pool", path)
		_, err := os.Stat(path)
		if err != nil {
			// We just ignore non-existing paths...
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			// ...but everything else is considered an error
			return nil, fmt.Errorf("could not load TLS certificate: %v", err)
		} else {
			f, err := ioutil.ReadFile(path)
			if err != nil {
				return nil, fmt.Errorf("failure to load TLS certificates from %s: %v", path, err)
			}
			if ok := pool.AppendCertsFromPEM(f); !ok {
				return nil, fmt.Errorf("invalid cert data in %s", path)
			}
		}
	}
	return pool, nil
}

// CreateServerTLSConfig will provide a TLS configuration for a server. It will
// either use a certificate and key provided at tlsCertPath and tlsKeyPath, or
// if these are not given, will generate a self-signed certificate valid for
// the specified list of hosts. If hosts is nil or empty, self-signed cert
// creation will be disabled.
func CreateServerTLSConfig(tlsCertPath, tlsKeyPath string, hosts []string) (*tls.Config, error) {
	var cert *tls.Certificate
	var err error

	tlsCertExists := false
	tlsKeyExists := false

	// If cert and key paths were specified, ensure they exist
	if tlsCertPath != "" && tlsKeyPath != "" {
		_, err = os.Stat(tlsCertPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Warnf("could not read TLS cert from %s: %v", tlsCertPath, err)
			}
		} else {
			tlsCertExists = true
		}

		_, err = os.Stat(tlsKeyPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Warnf("could not read TLS cert from %s: %v", tlsKeyPath, err)
			}
		} else {
			tlsKeyExists = true
		}
	}

	if !tlsCertExists || !tlsKeyExists {
		log.Infof("Generating self-signed gRPC TLS certificate for this session")
		c, err := GenerateX509KeyPair()
		if err != nil {
			return nil, err
		}
		cert = c
	} else {
		log.Infof("Loading gRPC TLS configuration from cert=%s and key=%s", tlsCertPath, tlsKeyPath)
		c, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
		if err != nil {
			return nil, fmt.Errorf("Unable to initalize gRPC TLS configuration with cert=%s and key=%s: %v", tlsCertPath, tlsKeyPath, err)
		}
		cert = &c
	}

	return &tls.Config{Certificates: []tls.Certificate{*cert}}, nil

}

func GetServerTLSConfigFromSecret(kubectlConfig kubernetes.Interface, tlsKubernetesSecretName string, tlsMinVersion uint16, namespace string) (*tls.Config, error) {

	ctx := context.Background()

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
		MinVersion:   uint16(tlsMinVersion),
	}, nil
}
