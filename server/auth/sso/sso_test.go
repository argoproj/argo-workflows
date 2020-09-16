package sso

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/coreos/go-oidc"
	"github.com/stretchr/testify/assert"
	testhttp "github.com/stretchr/testify/http"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const testNamespace = "argo"

var userInfoError error

type fakeOidcProvider struct{}

func (fakeOidcProvider) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{}
}

func (p fakeOidcProvider) UserInfo(context.Context, oauth2.TokenSource) (*oidc.UserInfo, error) {
	return &oidc.UserInfo{Subject: "my-sub"}, userInfoError
}

func fakeOidcFactory(ctx context.Context, issuer string) (providerInterface, error) {
	return fakeOidcProvider{}, nil
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
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	require.NoError(t, err)
	ssoObject := ssoInterface.(*sso)
	assert.Equal(t, "sso-client-id-value", ssoObject.config.ClientID)
	assert.Equal(t, "sso-client-secret-value", ssoObject.config.ClientSecret)
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

func TestSSO(t *testing.T) {
	fakeClient := fake.NewSimpleClientset(ssoConfigSecret).CoreV1().Secrets(testNamespace)
	config := Config{
		Issuer:       "https://test-issuer",
		ClientID:     getSecretKeySelector("argo-sso-secret", "client-id"),
		ClientSecret: getSecretKeySelector("argo-sso-secret", "client-secret"),
		RedirectURL:  "https://dummy",
	}
	ssoInterface, err := newSso(fakeOidcFactory, config, fakeClient, "/", false)
	sso := ssoInterface.(*sso)
	assert.NoError(t, err)
	t.Run("HandleRedirect", func(t *testing.T) {
		r := &http.Request{}
		r.URL, err = url.Parse("/")
		w := &testhttp.TestResponseWriter{}
		sso.HandleRedirect(w, r)
		assert.Equal(t, 302, w.StatusCode)
		assert.Regexp(t, "oauthState=.*; Expires=.*; HttpOnly; SameSite=Lax", w.Header().Get("Set-Cookie"))
		assert.NotEmpty(t, w.Header().Get("Location"))
	})
	t.Run("HandleCallback", func(t *testing.T) {
		t.Run("NoState", func(t *testing.T) {
			r := &http.Request{}
			r.URL, err = url.Parse("https://localhost:2746?state=my-state")
			w := &testhttp.TestResponseWriter{}
			sso.HandleCallback(w, r)
			assert.Equal(t, 400, w.StatusCode)
			assert.Contains(t, w.Output, "invalid state: http: named cookie not present")
		})
		t.Run("StateMismatch", func(t *testing.T) {
			r := &http.Request{Header: map[string][]string{"Cookie": {"oauthState=my-state"}}}
			r.URL, err = url.Parse("https://localhost:2746?state=wrong-state")
			w := &testhttp.TestResponseWriter{}
			sso.HandleCallback(w, r)
			assert.Equal(t, 401, w.StatusCode)
			assert.Contains(t, w.Output, "invalid state: does not match cookie value")
		})
		t.Run("FailedToExchangeToken", func(t *testing.T) {
			r := &http.Request{Header: map[string][]string{"Cookie": {"oauthState=my-state"}}}
			r.URL, err = url.Parse("https://localhost:2746?state=my-state&code=my-code")
			w := &testhttp.TestResponseWriter{}
			sso.HandleCallback(w, r)
			assert.Equal(t, 401, w.StatusCode)
			assert.Contains(t, w.Output, "failed to exchange token")
		})
	})
	t.Run("Authorize", func(t *testing.T) {
		t.Run("NotBase64Token", func(t *testing.T) {
			_, err := ssoInterface.Authorize(context.Background(), "garbage")
			assert.EqualError(t, err, "failed to decode encrypted access token: illegal base64 data at input byte 4")
		})
		t.Run("CorruptToken", func(t *testing.T) {
			key, err := generateKey()
			assert.NoError(t, err)
			encryptedAccessToken, err := encrypt(key, []byte("garbage"))
			assert.NoError(t, err)
			encodedEncryptedAccessToken := base64.StdEncoding.EncodeToString(encryptedAccessToken)
			_, err = ssoInterface.Authorize(context.Background(), encodedEncryptedAccessToken)
			assert.EqualError(t, err, "failed to decrypt encrypted access token: cipher: message authentication failed")
		})
		// set-up a garbage token
		encryptedAccessToken, err := encrypt(sso.cookieEncryptionKey, []byte("garbage"))
		assert.NoError(t, err)
		encodedEncryptedAccessToken := base64.StdEncoding.EncodeToString(encryptedAccessToken)
		t.Run("UserInfoErr", func(t *testing.T) {
			userInfoError = errors.New("my-error")
			defer func() { userInfoError = nil }()
			_, err := sso.Authorize(context.Background(), encodedEncryptedAccessToken)
			assert.EqualError(t, err, "failed to get user info: my-error")
		})
		t.Run("Successful", func(t *testing.T) {
			claimSet, err := sso.Authorize(context.Background(), encodedEncryptedAccessToken)
			if assert.NoError(t, err) {
				assert.Equal(t, "https://test-issuer", claimSet.Iss)
				assert.Equal(t, "my-sub", claimSet.Sub)
			}
		})
	})
}
