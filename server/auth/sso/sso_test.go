package sso

import (
	"context"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "argo"

type fakeOidcProvider struct {
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
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:               "https://test-issuer",
		IssuerAlias:          "",
		ClientID:             getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:         getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:          "https://dummy",
		CustomGroupClaimName: "argo_groups",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
	assert.Equal(t, "sso-client-secret-value", ssoObject.config.ClientSecret)
	assert.Equal(t, "argo_groups", ssoObject.customClaimName)
	assert.Equal(t, "", config.IssuerAlias)
	assert.Equal(t, 10*time.Hour, ssoObject.expiry)
}

func TestNewSsoWithIssuerAlias(t *testing.T) {
	// if there's an issuer alias present, the oidc provider will allow validation from either of the issuer or the issuerAlias.
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:               "https://test-issuer",
		IssuerAlias:          "https://test-issuer-alias",
		ClientID:             getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:         getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:          "https://dummy",
		CustomGroupClaimName: "argo_groups",
	}
	_, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
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

	fakeClient := fake.NewSimpleClientset(ssoConfigSecret, clientIDSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("other-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
}

func TestLoadSsoClientIdFromSecretNoKeyFails(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "nonexistent"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	_, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.Error(t, err)
	assert.Regexp(t, "key nonexistent missing in secret argo-sso-secret", err.Error())
}

func TestLoadSsoClientIdFromExistingSsoSecretFails(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)

	ctx := context.Background()
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
	_, err = newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.Error(t, err)
	assert.Regexp(t, "If you have already defined a Secret named sso, delete it and retry", err.Error())
}

func TestGetSessionExpiry(t *testing.T) {
	config := Config{
		SessionExpiry: metav1.Duration{Duration: 5 * time.Hour},
	}
	assert.Equal(t, 5*time.Hour, config.GetSessionExpiry())
}

func TestIsValidFinalRedirectUrl(t *testing.T) {
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
			assert.Equal(t, tc.expected, isValidFinalRedirectUrl(tc.url))
		})
	}
}
