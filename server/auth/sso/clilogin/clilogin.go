// Package clilogin is the handoff between `argo auth login` and argo-server's
// SSO endpoints.
//
// The CLI listens on a loopback port and opens the browser at
// /oauth2/redirect with three extra query parameters: the loopback URL, a
// random state, and an ephemeral P-256 public key. After the usual OIDC login,
// /oauth2/callback redirects the browser to the loopback URL with the state and
// the Argo token encrypted to that key. The server keeps no state, and the
// token never appears in plaintext in the browser's history.
package clilogin

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// Query parameters the CLI adds to /oauth2/redirect.
const (
	ParamRedirect = "cli_redirect"
	ParamState    = "cli_state"
	ParamKey      = "cli_key"
)

// Query parameters the server adds to the loopback redirect.
const (
	ResponseParamState = "state"
	ResponseParamToken = "token"
)

const (
	keyAlgorithm      = jose.ECDH_ES_A256KW
	contentEncryption = jose.A256GCM
	// maxParamLength bounds each CLI parameter, because the request is stored in
	// a cookie for the duration of the login.
	maxParamLength = 512
	minStateLength = 16
)

// Request is a CLI login request carried through the OIDC login.
type Request struct {
	RedirectURL string `json:"r"`
	State       string `json:"s"`
	Key         string `json:"k"`
}

// Payload is what the server hands to the CLI.
type Payload struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

// ParseRequest reads a CLI login request from /oauth2/redirect query
// parameters. It returns nil, nil when the request isn't a CLI login.
func ParseRequest(q url.Values) (*Request, error) {
	if !q.Has(ParamRedirect) && !q.Has(ParamState) && !q.Has(ParamKey) {
		return nil, nil
	}
	r := &Request{RedirectURL: q.Get(ParamRedirect), State: q.Get(ParamState), Key: q.Get(ParamKey)}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Validate checks that the request is safe to act on.
func (r *Request) Validate() error {
	for name, v := range map[string]string{ParamRedirect: r.RedirectURL, ParamState: r.State, ParamKey: r.Key} {
		if v == "" {
			return fmt.Errorf("%s is required", name)
		}
		if len(v) > maxParamLength {
			return fmt.Errorf("%s is longer than %d characters", name, maxParamLength)
		}
	}
	if len(r.State) < minStateLength {
		return fmt.Errorf("%s is shorter than %d characters", ParamState, minStateLength)
	}
	if err := ValidateRedirectURL(r.RedirectURL); err != nil {
		return err
	}
	_, err := r.publicKey()
	return err
}

// ValidateRedirectURL accepts only plain-HTTP loopback IP addresses with an
// explicit port, as described in RFC 8252 section 7.3. Host names, including
// "localhost", are rejected because they can resolve elsewhere.
func ValidateRedirectURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid %s: %w", ParamRedirect, err)
	}
	if u.Scheme != "http" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return fmt.Errorf("%s must be a plain http:// URL", ParamRedirect)
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("%s must use a loopback IP address", ParamRedirect)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("%s must have a port", ParamRedirect)
	}
	return nil
}

func (r *Request) publicKey() (*ecdsa.PublicKey, error) {
	var jwk jose.JSONWebKey
	if err := jwk.UnmarshalJSON([]byte(r.Key)); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", ParamKey, err)
	}
	key, ok := jwk.Key.(*ecdsa.PublicKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, fmt.Errorf("%s must be a P-256 public key", ParamKey)
	}
	return key, nil
}

// Encode serializes the request for storage in a cookie.
func (r *Request) Encode() (string, error) {
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeRequest reverses Encode and re-validates the request.
func DecodeRequest(s string) (*Request, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	r := &Request{}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, err
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Seal encrypts p to the request's key and returns the loopback URL to
// redirect the browser to.
func (r *Request) Seal(p Payload) (string, error) {
	key, err := r.publicKey()
	if err != nil {
		return "", err
	}
	encrypter, err := jose.NewEncrypter(contentEncryption, jose.Recipient{Algorithm: keyAlgorithm, Key: key}, nil)
	if err != nil {
		return "", err
	}
	plaintext, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	obj, err := encrypter.Encrypt(plaintext)
	if err != nil {
		return "", err
	}
	compact, err := obj.CompactSerialize()
	if err != nil {
		return "", err
	}
	u, err := url.Parse(r.RedirectURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set(ResponseParamState, r.State)
	q.Set(ResponseParamToken, compact)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Session is the CLI's side of one login attempt.
type Session struct {
	key   *ecdsa.PrivateKey
	state string
}

// NewSession creates a fresh key pair and state.
func NewSession() (*Session, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return &Session{key: key, state: base64.RawURLEncoding.EncodeToString(b)}, nil
}

// Params returns the /oauth2/redirect query parameters for this session.
func (s *Session) Params(redirectURL string) (url.Values, error) {
	jwk, err := jose.JSONWebKey{Key: &s.key.PublicKey}.MarshalJSON()
	if err != nil {
		return nil, err
	}
	return url.Values{ParamRedirect: {redirectURL}, ParamState: {s.state}, ParamKey: {string(jwk)}}, nil
}

// Open checks the state and decrypts the token from the loopback callback's
// query parameters.
func (s *Session) Open(q url.Values) (*Payload, error) {
	if subtle.ConstantTimeCompare([]byte(q.Get(ResponseParamState)), []byte(s.state)) != 1 {
		return nil, errors.New("state does not match this login")
	}
	obj, err := jose.ParseEncryptedCompact(q.Get(ResponseParamToken), []jose.KeyAlgorithm{keyAlgorithm}, []jose.ContentEncryption{contentEncryption})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}
	plaintext, err := obj.Decrypt(s.key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}
	p := &Payload{}
	if err := json.Unmarshal(plaintext, p); err != nil {
		return nil, fmt.Errorf("failed to read token: %w", err)
	}
	if p.Token == "" {
		return nil, errors.New("server returned an empty token")
	}
	return p, nil
}
