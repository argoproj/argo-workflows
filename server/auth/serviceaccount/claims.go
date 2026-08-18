package serviceaccount

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"

	"github.com/go-jose/go-jose/v4/jwt"
	"k8s.io/client-go/rest"

	"github.com/argoproj/argo-workflows/v4/server/auth/types"
)

func ClaimSetFor(restConfig *rest.Config) (*types.Claims, error) {
	username := restConfig.Username
	if username != "" {
		return &types.Claims{Claims: jwt.Claims{Subject: username}}, nil
	}

	if restConfig.BearerToken != "" || restConfig.BearerTokenFile != "" {
		return ClaimSetWithBearerToken(restConfig)
	}

	if restConfig.CertFile != "" || len(restConfig.CertData) > 0 {
		return ClaimSetWithX509(restConfig)
	}
	return nil, nil
}

func ClaimSetWithBearerToken(restConfig *rest.Config) (*types.Claims, error) {
	bearerToken := restConfig.BearerToken
	if bearerToken == "" {
		// should only ever be used for service accounts
		data, err := os.ReadFile(restConfig.BearerTokenFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read bearer token file: %w", err)
		}
		bearerToken = string(data)
	}

	parts := strings.SplitN(bearerToken, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("expected bearer token to be a JWT and therefore have 3 dot-delimited parts")
	}
	payload := parts[1]
	data, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode bearer token's JWT payload: %w", err)
	}

	claims := &types.Claims{}
	err = json.Unmarshal(data, &claims)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bearer token's JWT payload: %w", err)
	}

	// attempt to derive SA name and namespace from Subject
	// "system:serviceaccount:argo:jenkins" -> "argo", "jenkins"
	// note that the SA name can have a colon in it, although the rest cannot
	parts = strings.SplitN(claims.Subject, ":", 4)
	if len(parts) < 4 {
		return claims, nil
	}
	claims.ServiceAccountNamespace = parts[2]
	claims.ServiceAccountName = parts[3]

	return claims, nil
}

func ClaimSetWithX509(restConfig *rest.Config) (*types.Claims, error) {
	var cert *x509.Certificate
	var err error
	if len(restConfig.CertData) > 0 {
		// Decode certificate from memory data
		block, _ := pem.Decode(restConfig.CertData)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("failed to parse certificate PEM")
		}
		cert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
	} else {
		// Load certificate from file
		data, err := os.ReadFile(restConfig.CertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read certificate file: %w", err)
		}
		block, _ := pem.Decode(data)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("failed to parse certificate PEM")
		}
		cert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
	}

	if cert == nil {
		return nil, fmt.Errorf("failed to parse certificate")
	}

	// Extract username from CommonName (CN)
	username := cert.Subject.CommonName

	// Extract group information from Organization (O) fields
	var groups []string
	for _, org := range cert.Subject.Organization {
		if strings.HasPrefix(org, "system:") {
			groups = append(groups, org)
		}
	}

	// Construct claims object
	claims := &types.Claims{
		Claims: jwt.Claims{
			Subject: username,
			Issuer:  "kubernetes/cert",
		},
		Groups: groups,
	}

	return claims, nil
}
