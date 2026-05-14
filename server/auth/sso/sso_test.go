package sso

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/util/logging"
)

const testNamespace = "argo"

type fakeOidcProvider struct {
	//nolint:containedctx
	Ctx    context.Context
	Issuer string
}

func (fakeOidcProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{}
}

func (fakeOidcProvider) Verifier(config *oidc.Config) *oidc.IDTokenVerifier {
	return nil
}

func fakeOidcFactory(ctx context.Context, issuer string) (providerInterface, error) {
	return fakeOidcProvider{ctx, issuer}, nil
}

func getSecretKeySelector(secret, key string) apiv1.SecretKeySelector {
	return apiv1.SecretKeySelector{
		LocalObjectReference: apiv1.LocalObjectReference{
			Name: secret,
		},
		Key: key,
	}
}

var ssoConfigSecret = &apiv1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Namespace: testNamespace,
		Name:      "argo-sso-secret",
	},
	Type: apiv1.SecretTypeOpaque,
	Data: map[string][]byte{
		"client-id":     []byte("sso-client-id-value"),
		"client-secret": []byte("sso-client-secret-value"),
	},
}

func TestLoadSsoClientIdFromSecret(t *testing.T) {
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:               "https://test-issuer",
		IssuerAlias:          "",
		ClientID:             getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:         getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:          "https://dummy",
		CustomGroupClaimName: "argo_groups",
	}
	ssoInterface, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
	assert.Equal(t, "sso-client-secret-value", ssoObject.config.ClientSecret)
	assert.Equal(t, "argo_groups", ssoObject.customClaimName)
	assert.Empty(t, config.IssuerAlias)
	assert.Equal(t, 10*time.Hour, ssoObject.expiry)
}

func TestNewSsoWithIssuerAlias(t *testing.T) {
	// if there's an issuer alias present, the oidc provider will allow validation from either of the issuer or the issuerAlias.
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:               "https://test-issuer",
		IssuerAlias:          "https://test-issuer-alias",
		ClientID:             getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:         getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:          "https://dummy",
		CustomGroupClaimName: "argo_groups",
	}
	_, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
}
func TestLoadSsoClientIdFromDifferentSecret(t *testing.T) {
	clientIDSecret := &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      "other-secret",
		},
		Type: apiv1.SecretTypeOpaque,
		Data: map[string][]byte{
			"client-id": []byte("sso-client-id-value"),
		},
	}

	fakeClient := fake.NewClientset(ssoConfigSecret, clientIDSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("other-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	ssoInterface, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
}

func TestLoadSsoClientIdFromSecretNoKeyFails(t *testing.T) {
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "nonexistent"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	_, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, config, fakeClient, "/", false)
	require.Error(t, err)
	assert.Regexp(t, "key nonexistent missing in secret argo-sso-secret", err.Error())
}

func TestLoadSsoClientIdFromExistingSsoSecretFails(t *testing.T) {
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)

	ctx := logging.TestContext(t.Context())
	_, err := fakeClient.Create(ctx, &apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName},
		Data:       map[string][]byte{},
	}, metav1.CreateOptions{})
	require.NoError(t, err)

	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	_, err = newSso(logging.TestContext(t.Context()), fakeOidcFactory, config, fakeClient, "/", false)
	require.Error(t, err)
	assert.Regexp(t, "If you have already defined a Secret named sso, delete it and retry", err.Error())
}

func TestGetSessionExpiry(t *testing.T) {
	config := Config{
		SessionExpiry: metav1.Duration{Duration: 5 * time.Hour},
	}
	assert.Equal(t, 5*time.Hour, config.GetSessionExpiry())
}

func TestIsValidFinalRedirectURL(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected bool
	}{
		// Adapted from https://github.com/oauth2-proxy/oauth2-proxy/blob/ab448cf38e7c1f0740b3cc2448284775e39d9661/pkg/app/redirect/validator_test.go#L60-L116
		{"No Redirect", "", false},
		{"Single Slash", "/redirect", true},
		{"Single Slash with query parameters", "/redirect?foo=bar&baz=2", true},
		{"Double Slash (protocol-relative URL)", "//redirect", false},
		{"Absolute HTTP", "http://foo.bar/redirect", false},
		{"Absolute HTTP with subdomain", "http://baz.foo.bar/", false},
		{"Absolute HTTPS", "https://foo.bar/redirect", false},
		{"Absolute HTTPS Port and Domain", "https://evil.corp:3838/redirect", false},
		{"Escape Double Slash", "/\\evil.com", false},
		{"Space Single Slash", "/ /evil.com", false},
		{"Space Double Slash", "/ \\evil.com", false},
		{"Tab Single Slash", "/\t/evil.com", false},
		{"Tab Double Slash", "/\t\\evil.com", false},
		{"Vertical Tab Single Slash", "/\v/evil.com", false},
		{"Vertiacl Tab Double Slash", "/\v\\evil.com", false},
		{"New Line Single Slash", "/\n/evil.com", false},
		{"New Line Double Slash", "/\n\\evil.com", false},
		{"Carriage Return Single Slash", "/\r/evil.com", false},
		{"Carriage Return Double Slash", "/\r\\evil.com", false},
		{"Double Tab", "/\t/\t\\evil.com", false},
		{"Triple Tab 1", "/\t\t/\t/evil.com", false},
		{"Triple Tab 2", "/\t\t\\\t/evil.com", false},
		{"Quad Tab 1", "/\t\t/\t\t\\evil.com", false},
		{"Quad Tab 2", "/\t\t\\\t\t/evil.com", false},
		{"Relative Path", "/./\\evil.com", false},
		{"Relative Subpath", "/./../../\\evil.com", false},
		{"Missing Protocol Root Domain", "foo.bar/redirect", false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, isValidFinalRedirectURL(tc.url))
		})
	}
}

func TestEncodeDecodeStateCookieValue(t *testing.T) {
	testCases := []struct {
		name             string
		verifier         string
		finalRedirectURL string
	}{
		{"empty verifier (PKCE disabled)", "", "/workflows"},
		{"empty verifier and empty redirect", "", ""},
		{"verifier and redirect", "thisIsAVerifier_-1234", "/workflows/argo"},
		{"verifier and redirect with query", "v_eR1f1eR", "/workflows?foo=bar&baz=qux"},
		{"verifier only (defensive)", "v_eR1f1eR", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := encodeStateCookieValue(tc.verifier, tc.finalRedirectURL)
			gotVerifier, gotRedirect := decodeStateCookieValue(encoded)
			assert.Equal(t, tc.verifier, gotVerifier)
			assert.Equal(t, tc.finalRedirectURL, gotRedirect)
		})
	}
}

// TestDecodeStateCookieValueBackwardCompat verifies that a cookie value
// written by a pre-PKCE Argo version (containing only the redirect URL with
// no separator) is still decoded correctly: empty verifier, full redirect.
func TestDecodeStateCookieValueBackwardCompat(t *testing.T) {
	verifier, redirect := decodeStateCookieValue("/workflows")
	assert.Empty(t, verifier)
	assert.Equal(t, "/workflows", redirect)
}

func newSsoForTest(t *testing.T, mutate func(c *Config)) *sso {
	t.Helper()
	fakeClient := fake.NewClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	cfg := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://argo.example.com/oauth2/callback",
	}
	if mutate != nil {
		mutate(&cfg)
	}
	ssoInterface, err := newSso(logging.TestContext(t.Context()), fakeOidcFactory, cfg, fakeClient, "/", false)
	require.NoError(t, err)
	return ssoInterface.(*sso)
}

// TestPKCEEnabledByDefault verifies that PKCE is on unless explicitly opted
// out via InsecureSkipPKCE, matching the OAuth 2.0 Security BCP recommendation.
func TestPKCEEnabledByDefault(t *testing.T) {
	t.Run("default: enabled", func(t *testing.T) {
		s := newSsoForTest(t, nil)
		assert.True(t, s.pkceEnabled)
	})
	t.Run("InsecureSkipPKCE=true disables", func(t *testing.T) {
		s := newSsoForTest(t, func(c *Config) { c.InsecureSkipPKCE = true })
		assert.False(t, s.pkceEnabled)
	})
}

// TestHandleRedirectAddsPKCEChallenge verifies that when PKCE is enabled,
// HandleRedirect adds code_challenge and code_challenge_method=S256 to the
// authorization URL and stores the corresponding verifier in the state cookie.
func TestHandleRedirectAddsPKCEChallenge(t *testing.T) {
	s := newSsoForTest(t, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://argo.example.com/oauth2/redirect?redirect=/workflows", nil)
	s.HandleRedirect(rec, req)

	require.Equal(t, http.StatusFound, rec.Code, "expected redirect to IdP")

	authURL, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)

	q := authURL.Query()
	assert.NotEmpty(t, q.Get("code_challenge"), "code_challenge must be present when PKCE is enabled")
	assert.Equal(t, "S256", q.Get("code_challenge_method"), "S256 is the only safe challenge method")
	state := q.Get("state")
	require.NotEmpty(t, state)

	// The state cookie value must contain the verifier paired with the redirect URL.
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == state {
			stateCookie = c
			break
		}
	}
	require.NotNil(t, stateCookie, "state cookie must be set")
	verifier, finalRedirect := decodeStateCookieValue(stateCookie.Value)
	assert.NotEmpty(t, verifier, "verifier must be stored in state cookie")
	assert.Equal(t, "/workflows", finalRedirect)
	// RFC 7636 §4.1: verifier is 43–128 chars from the unreserved set.
	assert.GreaterOrEqual(t, len(verifier), 43)
	assert.LessOrEqual(t, len(verifier), 128)
	// Cookie hardening attributes preserved.
	assert.True(t, stateCookie.HttpOnly)
	assert.Equal(t, http.SameSiteLaxMode, stateCookie.SameSite)
}

// TestHandleRedirectOmitsPKCEWhenDisabled verifies opt-out preserves the
// pre-PKCE wire format: no code_challenge param, plain redirect-URL cookie value.
func TestHandleRedirectOmitsPKCEWhenDisabled(t *testing.T) {
	s := newSsoForTest(t, func(c *Config) { c.InsecureSkipPKCE = true })

	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://argo.example.com/oauth2/redirect?redirect=/workflows", nil)
	s.HandleRedirect(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)
	authURL, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	q := authURL.Query()
	assert.Empty(t, q.Get("code_challenge"))
	assert.Empty(t, q.Get("code_challenge_method"))

	state := q.Get("state")
	cookies := rec.Result().Cookies()
	var stateCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == state {
			stateCookie = c
			break
		}
	}
	require.NotNil(t, stateCookie)
	// No separator → cookie value is exactly the redirect URL (back-compat wire format).
	assert.Equal(t, "/workflows", stateCookie.Value)
	assert.NotContains(t, stateCookie.Value, stateCookieSeparator)
}
