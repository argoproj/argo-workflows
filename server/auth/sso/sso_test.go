package sso

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/coreos/go-oidc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "argo"

type fakeOidcProvider struct{}

func (fakeOidcProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{}
}

func (fakeOidcProvider) Verifier(config *oidc.Config) *oidc.IDTokenVerifier {
	return nil
}

func fakeOidcFactory(ctx context.Context, issuer string) (providerInterface, error) {
	return fakeOidcProvider{}, nil
}

func encodeSecretValue(value string) []byte {
	src := []byte(value)
	dst := make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	return dst
}
func decodeSecretValue(t *testing.T, value string) string {
	result, err := base64.StdEncoding.DecodeString(value)
	require.NoError(t, err)
	return string(result)
}

func getSecretKeySelector(secret, key string) *apiv1.SecretKeySelector {
	return &apiv1.SecretKeySelector{
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
		"client-id":     encodeSecretValue("sso-client-id-value"),
		"client-secret": encodeSecretValue("sso-client-secret-value"),
	},
}

func TestLoadSsoClientIdDirectly(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:         "https://test-issuer",
		ClientID:       "sso-client-id-direct-value",
		ClientIDSecret: nil,
		ClientSecret:   *getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:    "https://dummy",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-direct-value", ssoObject.config.ClientID)
	assert.Equal(t, "sso-client-secret-value", decodeSecretValue(t, ssoObject.config.ClientSecret))
}

func TestLoadSsoClientIdFromSecret(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:         "https://test-issuer",
		ClientID:       "",
		ClientIDSecret: getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:   *getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:    "https://dummy",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
	assert.Equal(t, "sso-client-secret-value", decodeSecretValue(t, ssoObject.config.ClientSecret))
}

func TestLoadSsoClientIdFromSecretAndDirectFails(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:         "https://test-issuer",
		ClientID:       "sso-client-id-direct-value",
		ClientIDSecret: getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret:   *getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:    "https://dummy",
	}
	_, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.Error(t, err)
	assert.Regexp(t, "only one of .* must be specified", err.Error())
}
