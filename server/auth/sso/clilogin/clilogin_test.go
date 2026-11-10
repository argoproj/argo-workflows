package clilogin

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"maps"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const loopback = "http://127.0.0.1:8085/callback"

func newRequest(t *testing.T, s *Session, redirect string) *Request {
	t.Helper()
	params, err := s.Params(redirect)
	require.NoError(t, err)
	r, err := ParseRequest(params)
	require.NoError(t, err)
	require.NotNil(t, r)
	return r
}

func TestRoundTrip(t *testing.T) {
	s, err := NewSession()
	require.NoError(t, err)
	r := newRequest(t, s, loopback)

	encoded, err := r.Encode()
	require.NoError(t, err)
	assert.Less(t, len(encoded), 1024, "the request must fit comfortably in a cookie")
	decoded, err := DecodeRequest(encoded)
	require.NoError(t, err)
	assert.Equal(t, r, decoded)

	expiry := time.Now().Add(time.Hour).Truncate(time.Second).UTC()
	location, err := decoded.Seal(Payload{Token: "Bearer v2:abc", Expiry: expiry})
	require.NoError(t, err)
	u, err := url.Parse(location)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:8085", u.Host)
	assert.Equal(t, "/callback", u.Path)
	assert.NotContains(t, location, "abc", "the token must not be in plaintext")

	p, err := s.Open(u.Query())
	require.NoError(t, err)
	assert.Equal(t, "Bearer v2:abc", p.Token)
	assert.True(t, expiry.Equal(p.Expiry))
}

func TestOpenRejectsWrongStateOrKey(t *testing.T) {
	s, err := NewSession()
	require.NoError(t, err)
	location, err := newRequest(t, s, loopback).Seal(Payload{Token: "t"})
	require.NoError(t, err)
	u, err := url.Parse(location)
	require.NoError(t, err)

	q := u.Query()
	q.Set(ResponseParamState, "wrong")
	_, err = s.Open(q)
	require.ErrorContains(t, err, "state")

	other, err := NewSession()
	require.NoError(t, err)
	q = u.Query()
	q.Set(ResponseParamState, other.state)
	_, err = other.Open(q)
	require.ErrorContains(t, err, "decrypt")
}

func TestParseRequestNotCLI(t *testing.T) {
	r, err := ParseRequest(url.Values{"redirect": {"/workflows"}})
	require.NoError(t, err)
	assert.Nil(t, r)
}

func TestParseRequestRejects(t *testing.T) {
	s, err := NewSession()
	require.NoError(t, err)
	valid, err := s.Params(loopback)
	require.NoError(t, err)

	jwk := func(key any) string {
		b, marshalErr := jose.JSONWebKey{Key: key}.MarshalJSON()
		require.NoError(t, marshalErr)
		return string(b)
	}
	p384, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	rsaKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	tests := map[string]struct {
		param, value, want string
	}{
		"missing redirect":     {ParamRedirect, "", "required"},
		"https redirect":       {ParamRedirect, "https://127.0.0.1:8085/", "plain http"},
		"remote host":          {ParamRedirect, "http://evil.example.com:8085/", "loopback"},
		"localhost name":       {ParamRedirect, "http://localhost:8085/", "loopback"},
		"no port":              {ParamRedirect, "http://127.0.0.1/", "port"},
		"user info":            {ParamRedirect, "http://evil@127.0.0.1:8085/", "plain http"},
		"fragment":             {ParamRedirect, "http://127.0.0.1:8085/#x", "plain http"},
		"short state":          {ParamState, "abc", "shorter"},
		"long state":           {ParamState, strings.Repeat("a", maxParamLength+1), "longer"},
		"not a key":            {ParamKey, "{}", ParamKey},
		"P-384 key":            {ParamKey, jwk(&p384.PublicKey), "P-256"},
		"RSA key":              {ParamKey, jwk(&rsaKey.PublicKey), "P-256"},
		"private key supplied": {ParamKey, jwk(s.key), "P-256"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			q := url.Values{}
			maps.Copy(q, valid)
			q.Set(tt.param, tt.value)
			_, err := ParseRequest(q)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestValidateRedirectURLIPv6(t *testing.T) {
	require.NoError(t, ValidateRedirectURL("http://[::1]:8085/callback"))
}
