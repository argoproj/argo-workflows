package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"strings"
)

// XHubConfig holds configuration for X-Hub signature validation.
type XHubConfig struct {
	// HashAlgorithm specifies the hash algorithm to use (e.g., "sha1", "sha256").
	// Defaults to "sha256" if empty.
	HashAlgorithm string
	// HeaderName specifies the header containing the signature.
	// Defaults to "X-Hub-Signature-256" if empty.
	HeaderName string
	// Encoding specifies the signature encoding ("hex" or "base64").
	// Defaults to "hex" if empty.
	Encoding string
}

// xHubMatch validates X-Hub-Signature headers using HMAC.
// This provides a generic webhook authentication that works with any platform following
// the standard X-Hub-Signature format (Gitea, Forgejo, and others).
func xHubMatch(secret string, r *http.Request, config *XHubConfig) bool {

	if len(secret) == 0 {
		return false
	}

	// Apply defaults
	hashAlgorithm := "sha256"
	headerName := "X-Hub-Signature-256"
	encoding := "hex"
	if config != nil {
		if config.HashAlgorithm != "" {
			hashAlgorithm = config.HashAlgorithm
		}
		if config.HeaderName != "" {
			headerName = config.HeaderName
		}
		if config.Encoding != "" {
			encoding = config.Encoding
		}
	}

	signature := r.Header.Get(headerName)
	if len(signature) == 0 {
		return false
	}

	payload, err := io.ReadAll(r.Body)
	if err != nil || len(payload) == 0 {
		return false
	}

	// Trim the hash prefix (e.g., "sha256=" or "sha1=")
	signature = strings.TrimPrefix(signature, hashAlgorithm+"=")

	// Select hash function based on algorithm
	var hashFunc func() hash.Hash
	switch hashAlgorithm {
	case "sha1":
		hashFunc = sha1.New
	case "sha256":
		hashFunc = sha256.New
	case "sha384":
		hashFunc = sha512.New384
	case "sha512":
		hashFunc = sha512.New
	default:
		return false
	}

	mac := hmac.New(hashFunc, []byte(secret))
	_, _ = mac.Write(payload)
	expectedMAC := mac.Sum(nil)

	// Decode signature based on encoding
	var signatureBytes []byte

	switch encoding {
	case "base64":
		signatureBytes, err = base64.StdEncoding.DecodeString(signature)
	case "hex":
		signatureBytes, err = hex.DecodeString(signature)
	default:
		return false
	}
	if err != nil {
		return false
	}

	return hmac.Equal(signatureBytes, expectedMAC)

}
