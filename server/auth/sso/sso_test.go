package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/server/auth/types"
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

func TestAuthorizeEncryptedToken(t *testing.T) {
	ssoObject := newTestSso(t)
	raw, err := jwt.Encrypted(ssoObject.encrypter).Claims(newTestClaims()).Serialize()
	require.NoError(t, err)

	claims, err := ssoObject.Authorize(Prefix + raw)
	require.NoError(t, err)
	assert.Equal(t, "test-subject", claims.Subject)
	assert.Equal(t, []string{"test-group"}, claims.Groups)
}

// Tokens from v4.1.x used a nested RS256 JWS inside an RSA-OAEP-256 JWE
// (and pre-4.1 an RSA-OAEP-256 JWE alone); both must be rejected at parse
// so that stale cookies force a fresh login rather than a server error.
func TestAuthorizeLegacyAsymmetricTokenFails(t *testing.T) {
	ssoObject := newTestSso(t)
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: privateKey}, nil)
	require.NoError(t, err)
	encrypterOptions := (&jose.EncrypterOptions{Compression: jose.DEFLATE}).WithContentType("JWT")
	legacyEncrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.RSA_OAEP_256, Key: privateKey.Public()}, encrypterOptions)
	require.NoError(t, err)
	raw, err := jwt.SignedAndEncrypted(signer, legacyEncrypter).Claims(newTestClaims()).Serialize()
	require.NoError(t, err)

	claims, err := ssoObject.Authorize(Prefix + raw)
	require.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorContains(t, err, "failed to parse encrypted token")
}

func TestAuthorizeTokenEncryptedWithOtherKeyFails(t *testing.T) {
	ssoObject := newTestSso(t)
	otherSso := newTestSso(t)
	raw, err := jwt.Encrypted(otherSso.encrypter).Claims(newTestClaims()).Serialize()
	require.NoError(t, err)

	claims, err := ssoObject.Authorize(Prefix + raw)
	require.Error(t, err)
	assert.Nil(t, claims)
	assert.ErrorContains(t, err, "failed to decrypt token")
}

// A large but realistic claims set (long email, many groups) must serialize
// to a cookie under the 4096-byte browser limit (RFC 6265).
// https://github.com/argoproj/argo-workflows/issues/16744
func TestTokenFitsInCookie(t *testing.T) {
	ssoObject := newTestSso(t)
	groups := make([]string, 60)
	for i := range groups {
		groups[i] = fmt.Sprintf("department-%02d-engineering-owners@example-organization.com", i)
	}
	claims := &types.Claims{
		Claims: jwt.Claims{
			Issuer:  issuer,
			Subject: "00u1abcdefghijklmnop5d7",
			Expiry:  jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		Groups:            groups,
		Email:             "somebody.with-a-long-name@example-organization.com",
		EmailVerified:     true,
		Name:              "Somebody With-A-Long-Name",
		PreferredUsername: "somebody.with-a-long-name@example-organization.com",
	}
	raw, err := jwt.Encrypted(ssoObject.encrypter).Claims(claims).Serialize()
	require.NoError(t, err)
	cookie := "authorization=" + Prefix + raw
	assert.Less(t, len(cookie), 4096)

	roundTripped, err := ssoObject.Authorize(Prefix + raw)
	require.NoError(t, err)
	assert.Equal(t, groups, roundTripped.Groups)
}

func newTestSso(t *testing.T) *sso {
	t.Helper()
	encryptionKey := make([]byte, 32)
	_, err := rand.Read(encryptionKey)
	require.NoError(t, err)
	encrypter, err := jose.NewEncrypter(jose.A256GCM, jose.Recipient{Algorithm: jose.DIRECT, Key: encryptionKey}, &jose.EncrypterOptions{Compression: jose.DEFLATE})
	require.NoError(t, err)
	return &sso{encryptionKey: encryptionKey, encrypter: encrypter}
}

func newTestClaims() *types.Claims {
	return &types.Claims{
		Claims: jwt.Claims{
			Issuer:  issuer,
			Subject: "test-subject",
			Expiry:  jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		Groups: []string{"test-group"},
	}
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
