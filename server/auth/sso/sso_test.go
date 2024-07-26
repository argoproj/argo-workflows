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
