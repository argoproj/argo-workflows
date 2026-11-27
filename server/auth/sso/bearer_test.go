package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/config"
	"github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const (
	bearerTestIssuer = "https://idp.example.com/tenant/v2.0"
	testKeyID        = "key-1"
)

var testRSAKeys = sync.OnceValue(func() [2]*rsa.PrivateKey {
	var keys [2]*rsa.PrivateKey
	for i := range keys {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}
		keys[i] = key
	}
	return keys
})

type jwksProvider struct {
	jwksURL string
}

func (jwksProvider) Endpoint() oauth2.Endpoint { return oauth2.Endpoint{} }

func (jwksProvider) Verifier(*oidc.Config) *oidc.IDTokenVerifier { return nil }

func (p jwksProvider) Claims(v any) error {
	metadata, err := json.Marshal(map[string]any{"jwks_uri": p.jwksURL, "id_token_signing_alg_values_supported": []string{"RS256"}})
	if err != nil {
		return err
	}
	return json.Unmarshal(metadata, v)
}

type publishedKey struct {
	key *rsa.PrivateKey
	kid string
	use string
}

type jwksServer struct {
	*httptest.Server
	current atomic.Pointer[publishedKey]
	status  atomic.Int32
	delay   atomic.Int64
	hits    atomic.Int32
}

func newJWKSServer(t *testing.T, key *rsa.PrivateKey) *jwksServer {
	t.Helper()
	srv := &jwksServer{}
	srv.current.Store(&publishedKey{key, testKeyID, "sig"})
	srv.status.Store(http.StatusOK)
	srv.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		srv.hits.Add(1)
		time.Sleep(time.Duration(srv.delay.Load()))
		if status := int(srv.status.Load()); status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		p := srv.current.Load()
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &p.key.PublicKey, KeyID: p.kid, Use: p.use}}})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newBearerSSO(ctx context.Context, t *testing.T, jwksURL string, c Config) *sso {
	t.Helper()
	c.Issuer = bearerTestIssuer
	c.ClientID = getSecretKeySelector("argo-sso-secret", "client-id")
	c.ClientSecret = getSecretKeySelector("argo-sso-secret", "client-secret")
	factory := func(context.Context, string) (providerInterface, error) {
		return jwksProvider{jwksURL: jwksURL}, nil
	}
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	s, err := newSso(ctx, factory, c, fakeClient, "/", false)
	require.NoError(t, err)
	return s.(*sso)
}

func signToken(t *testing.T, key any, kid string, alg jose.SignatureAlgorithm, claims map[string]any) string {
	t.Helper()
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: alg, Key: jose.JSONWebKey{Key: key, KeyID: kid}}, (&jose.SignerOptions{}).WithType("JWT"))
	require.NoError(t, err)
	raw, err := jwt.Signed(signer).Claims(claims).Serialize()
	require.NoError(t, err)
	return raw
}

func validClaims() map[string]any {
	return map[string]any{
		"iss":   bearerTestIssuer,
		"aud":   "sso-client-id-value",
		"sub":   "machine-id",
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"email": "machine@example.com",
	}
}

func withClaim(name string, value any) map[string]any {
	c := validClaims()
	if value == nil {
		delete(c, name)
	} else {
		c[name] = value
	}
	return c
}

func enabled() *config.SSOBearerTokenConfig {
	return &config.SSOBearerTokenConfig{Enabled: true}
}

func TestAuthorizeBearerRouting(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	key := testRSAKeys()[0]
	srv := newJWKSServer(t, key)
	token := signToken(t, key, testKeyID, jose.RS256, validClaims())
	payload := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }

	s := newBearerSSO(ctx, t, srv.URL, Config{BearerToken: enabled()})
	tests := []struct {
		name          string
		authorization string
		handled       bool
	}{
		{"JWT from the issuer", "Bearer " + token, true},
		{"HMAC JWT from the issuer", "Bearer " + signToken(t, []byte("01234567890123456789012345678901"), testKeyID, jose.HS256, validClaims()), true},
		{"JWT from another issuer", "Bearer " + signToken(t, key, testKeyID, jose.RS256, withClaim("iss", "https://kubernetes.default.svc")), false},
		{"Argo session token", Prefix + "a.b.c.d.e", false},
		{"Opaque token", "Bearer k8s-token", false},
		{"Basic auth", "Basic dXNlcjpwYXNz", false},
		{"Payload not base64", "Bearer a.!!!.c", false},
		{"Payload not JSON", "Bearer a." + payload("not json") + ".c", false},
		{"Issuer not a string", "Bearer a." + payload(`{"iss":1}`) + ".c", false},
		{"No space after Bearer", "Bearer" + token, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, handled, _ := s.AuthorizeBearer(ctx, tc.authorization)
			assert.Equal(t, tc.handled, handled)
		})
	}

	t.Run("Disabled", func(t *testing.T) {
		s := newBearerSSO(ctx, t, srv.URL, Config{})
		claims, handled, err := s.AuthorizeBearer(ctx, "Bearer "+token)
		assert.False(t, handled)
		assert.Nil(t, claims)
		require.NoError(t, err)
	})

	t.Run("IssuerAlias", func(t *testing.T) {
		s := newBearerSSO(ctx, t, srv.URL, Config{IssuerAlias: "https://alias.example.com", BearerToken: enabled()})
		_, handled, err := s.AuthorizeBearer(ctx, "Bearer "+signToken(t, key, testKeyID, jose.RS256, withClaim("iss", "https://alias.example.com")))
		assert.True(t, handled)
		require.NoError(t, err)
		_, handled, _ = s.AuthorizeBearer(ctx, "Bearer "+token)
		assert.False(t, handled)
	})
}

func TestAuthorizeBearer(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	key, otherKey := testRSAKeys()[0], testRSAKeys()[1]
	srv := newJWKSServer(t, key)
	authorize := func(s *sso, token string) (*types.Claims, error) {
		t.Helper()
		claims, handled, err := s.AuthorizeBearer(ctx, "Bearer "+token)
		require.True(t, handled)
		return claims, err
	}

	t.Run("Valid token without groups", func(t *testing.T) {
		s := newBearerSSO(ctx, t, srv.URL, Config{BearerToken: enabled()})
		claims, err := authorize(s, signToken(t, key, testKeyID, jose.RS256, withClaim("service_account_name", "admin")))
		require.NoError(t, err)
		assert.Equal(t, "machine-id", claims.Subject)
		assert.Equal(t, bearerTestIssuer, claims.Issuer)
		assert.Equal(t, "machine@example.com", claims.Email)
		assert.NotNil(t, claims.Groups)
		assert.Empty(t, claims.Groups)
		assert.Empty(t, claims.ServiceAccountName, "the token must not choose the service account")
	})

	groupTests := []struct {
		name     string
		config   Config
		claim    string
		groups   []string
		expected []string
	}{
		{"Groups are filtered", Config{FilterGroupsRegex: []string{"^argo-"}}, "groups", []string{"argo-readers", "unrelated"}, []string{"argo-readers"}},
		{"No group matches the filter", Config{FilterGroupsRegex: []string{"^argo-"}}, "groups", []string{"unrelated"}, []string{}},
		{"Custom group claim", Config{CustomGroupClaimName: "roles"}, "roles", []string{"Workflows.Read"}, []string{"Workflows.Read"}},
		{"Custom group claim missing", Config{CustomGroupClaimName: "roles"}, "groups", []string{"ignored"}, []string{}},
	}
	for _, tc := range groupTests {
		t.Run(tc.name, func(t *testing.T) {
			tc.config.BearerToken = enabled()
			s := newBearerSSO(ctx, t, srv.URL, tc.config)
			claims, err := authorize(s, signToken(t, key, testKeyID, jose.RS256, withClaim(tc.claim, tc.groups)))
			require.NoError(t, err)
			assert.Equal(t, tc.expected, claims.Groups)
		})
	}

	t.Run("Allowed audiences", func(t *testing.T) {
		s := newBearerSSO(ctx, t, srv.URL, Config{BearerToken: &config.SSOBearerTokenConfig{Enabled: true, AllowedAudiences: []string{"api://argo"}}})
		_, err := authorize(s, signToken(t, key, testKeyID, jose.RS256, withClaim("aud", []string{"other", "api://argo"})))
		require.NoError(t, err)
		_, err = authorize(s, signToken(t, key, testKeyID, jose.RS256, validClaims()))
		require.Error(t, err, "the client ID is no longer accepted once allowedAudiences is set")
	})

	unsigned := func() string {
		body, err := json.Marshal(validClaims())
		require.NoError(t, err)
		return base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`)) + "." + base64.RawURLEncoding.EncodeToString(body) + "."
	}()
	s := newBearerSSO(ctx, t, srv.URL, Config{BearerToken: enabled()})
	rejected := []struct {
		name  string
		token string
	}{
		{"Wrong audience", signToken(t, key, testKeyID, jose.RS256, withClaim("aud", "another-app"))},
		{"Missing audience", signToken(t, key, testKeyID, jose.RS256, withClaim("aud", nil))},
		{"Wrong issuer", signToken(t, key, testKeyID, jose.RS256, withClaim("iss", "https://evil.example.com"))},
		{"Expired", signToken(t, key, testKeyID, jose.RS256, withClaim("exp", time.Now().Add(-time.Minute).Unix()))},
		{"Not valid yet", signToken(t, key, testKeyID, jose.RS256, withClaim("nbf", time.Now().Add(time.Hour).Unix()))},
		{"Signed with another key", signToken(t, otherKey, testKeyID, jose.RS256, validClaims())},
		{"HMAC signed", signToken(t, []byte("01234567890123456789012345678901"), testKeyID, jose.HS256, validClaims())},
		{"Unsigned", unsigned},
	}
	for _, tc := range rejected {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.verifyBearer(ctx, tc.token)
			require.Error(t, err)
		})
	}
}

func TestNewSsoBearerTokenRequiresJWKS(t *testing.T) {
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	_, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, Config{
		Issuer:       bearerTestIssuer,
		ClientID:     getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		BearerToken:  enabled(),
	}, fakeClient, "/", false)
	require.ErrorContains(t, err, "jwks_uri")
}

func TestAuthorizeBearerSigningAlgs(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	key := testRSAKeys()[0]
	srv := newJWKSServer(t, key)
	s := newBearerSSO(ctx, t, srv.URL, Config{BearerToken: enabled()})
	for _, alg := range []jose.SignatureAlgorithm{jose.RS256, jose.RS384, jose.PS256} {
		t.Run(string(alg), func(t *testing.T) {
			_, _, err := s.AuthorizeBearer(ctx, "Bearer "+signToken(t, key, testKeyID, alg, validClaims()))
			require.NoError(t, err, "the provider only advertises RS256 for ID tokens, which doesn't restrict access tokens")
		})
	}
}

type fakeClock struct{ nanos atomic.Int64 }

func (c *fakeClock) now() time.Time { return time.Unix(0, c.nanos.Load()) }

func (c *fakeClock) add(d time.Duration) { c.nanos.Add(int64(d)) }

func TestJWKSKeySet(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	key, otherKey := testRSAKeys()[0], testRSAKeys()[1]
	good := signToken(t, key, testKeyID, jose.RS256, validClaims())
	forgedKnownKid := signToken(t, otherKey, testKeyID, jose.RS256, validClaims())
	unknownKid := signToken(t, otherKey, "unknown", jose.RS256, validClaims())

	newKeySet := func(t *testing.T, srv *jwksServer) (*jwksKeySet, *fakeClock) {
		t.Helper()
		keySet := newJWKSKeySet(srv.URL, http.DefaultClient)
		clock := &fakeClock{}
		clock.nanos.Store(time.Now().UnixNano())
		keySet.now = clock.now
		return keySet, clock
	}
	waitForFetch := func(keySet *jwksKeySet) {
		_, _, _ = keySet.fetches.Do("", func() (any, error) { return nil, nil })
	}

	t.Run("Bad signature with a known key ID doesn't refetch", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		keySet, _ := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		for range 10 {
			_, err := keySet.VerifySignature(ctx, forgedKnownKid)
			require.Error(t, err)
		}
		assert.Equal(t, int32(1), srv.hits.Load())
	})

	t.Run("Unknown key IDs refetch at most once per interval", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		keySet, clock := newKeySet(t, srv)
		for range 10 {
			_, err := keySet.VerifySignature(ctx, unknownKid)
			require.Error(t, err)
		}
		assert.Equal(t, int32(1), srv.hits.Load())
		clock.add(jwksMinRefreshInterval)
		_, err := keySet.VerifySignature(ctx, unknownKid)
		require.Error(t, err)
		assert.Equal(t, int32(2), srv.hits.Load())
	})

	t.Run("Concurrent requests share one fetch", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		srv.delay.Store(int64(50 * time.Millisecond))
		keySet, _ := newKeySet(t, srv)
		var wg sync.WaitGroup
		for range 20 {
			wg.Go(func() {
				_, err := keySet.VerifySignature(ctx, good)
				assert.NoError(t, err)
			})
		}
		wg.Wait()
		assert.Equal(t, int32(1), srv.hits.Load())
	})

	t.Run("Cancelled request doesn't cancel the fetch", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		srv.delay.Store(int64(50 * time.Millisecond))
		keySet, _ := newKeySet(t, srv)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, err := keySet.VerifySignature(cancelled, good)
		require.ErrorIs(t, err, context.Canceled)
		waitForFetch(keySet)
		_, err = keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		assert.Equal(t, int32(1), srv.hits.Load())
	})

	t.Run("Failed fetch is retried after a short interval", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		srv.status.Store(http.StatusInternalServerError)
		keySet, clock := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.ErrorContains(t, err, "500")
		srv.status.Store(http.StatusOK)
		_, err = keySet.VerifySignature(ctx, good)
		require.ErrorContains(t, err, "500", "the fetch failure is reported until the retry")
		clock.add(jwksRetryInterval)
		_, err = keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		assert.Equal(t, int32(2), srv.hits.Load())
	})

	t.Run("Rotated key is picked up", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		keySet, clock := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		srv.current.Store(&publishedKey{otherKey, "rotated", "sig"})
		clock.add(jwksMinRefreshInterval)
		_, err = keySet.VerifySignature(ctx, signToken(t, otherKey, "rotated", jose.RS256, validClaims()))
		require.NoError(t, err)
	})

	t.Run("Expired keys are refreshed in the background", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		keySet, clock := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		srv.current.Store(&publishedKey{otherKey, "rotated", "sig"})
		srv.delay.Store(int64(50 * time.Millisecond))
		clock.add(jwksMaxAge)
		_, err = keySet.VerifySignature(ctx, good)
		require.NoError(t, err, "the cached key answers without waiting for the fetch")
		waitForFetch(keySet)
		_, err = keySet.VerifySignature(ctx, good)
		require.Error(t, err, "the removed key is no longer trusted")
	})

	t.Run("Encryption keys are ignored", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		srv.current.Store(&publishedKey{key, testKeyID, "enc"})
		keySet, _ := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.Error(t, err)
	})

	t.Run("Cached keys keep working when the provider is down", func(t *testing.T) {
		srv := newJWKSServer(t, key)
		keySet, clock := newKeySet(t, srv)
		_, err := keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		srv.status.Store(http.StatusServiceUnavailable)
		clock.add(jwksMaxAge)
		_, err = keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		waitForFetch(keySet)
		_, err = keySet.VerifySignature(ctx, good)
		require.NoError(t, err)
		assert.Equal(t, int32(2), srv.hits.Load())
	})
}
