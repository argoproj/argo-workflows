package apiserver

import (
"crypto/rand"
"crypto/rsa"
"crypto/x509"
"crypto/x509/pkix"
"encoding/pem"
"math/big"
"net"
"time"
)

func generateSelfSignedCert(hosts []string, ips []net.IP) ([]byte, []byte, error) {
priv, err := rsa.GenerateKey(rand.Reader, 2048)
if err != nil {
return nil, nil, err
}

notBefore := time.Now()
notAfter := notBefore.Add(365 * 24 * time.Hour)

serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
if err != nil {
return nil, nil, err
}

template := x509.Certificate{
SerialNumber: serialNumber,
Subject: pkix.Name{
Organization: []string{"Argo Workflows"},
},
NotBefore:             notBefore,
NotAfter:              notAfter,
KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
BasicConstraintsValid: true,
DNSNames:              hosts,
IPAddresses:           ips,
}

derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
if err != nil {
return nil, nil, err
}

certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

return certPEM, keyPEM, nil
}
