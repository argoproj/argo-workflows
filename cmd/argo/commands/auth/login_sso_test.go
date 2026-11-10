package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/argoproj/argo-workflows/v4/server/auth/sso"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso/clilogin"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// These tests run the whole SSO login over HTTP: a fake OIDC provider, the
// real Argo Server SSO handlers (including go-oidc's ID token verification),
// the CLI's login code, and a cookie-keeping HTTP client as the browser.

const testClientID = "argo-cli-test"

// fakeIDP is a minimal OIDC provider that logs in one user without asking.
type fakeIDP struct {
	*httptest.Server
	key *rsa.PrivateKey
}

func newFakeIDP(t *testing.T) *fakeIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	idp := &fakeIDP{key: key}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"issuer":                                idp.URL,
			"authorization_endpoint":                idp.URL + "/auth",
			"token_endpoint":                        idp.URL + "/token",
			"jwks_uri":                              idp.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})
	mux.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, testClientID, q.Get("client_id"))
		redirect, err := url.Parse(q.Get("redirect_uri"))
		if !assert.NoError(t, err) {
			return
		}
		rq := redirect.Query()
		rq.Set("code", "test-code")
		rq.Set("state", q.Get("state"))
		redirect.RawQuery = rq.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, r.ParseForm()) || !assert.Equal(t, "test-code", r.PostForm.Get("code")) {
			http.Error(w, "bad code", http.StatusBadRequest)
			return
		}
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: idp.key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "k1"))
		if !assert.NoError(t, err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		idToken, err := jwt.Signed(signer).Claims(map[string]any{
			"iss":    idp.URL,
			"aud":    testClientID,
			"sub":    "alice",
			"email":  "alice@example.com",
			"groups": []string{"data-eng"},
			"iat":    time.Now().Unix(),
			"exp":    time.Now().Add(time.Hour).Unix(),
		}).Serialize()
		if !assert.NoError(t, err) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(t, w, map[string]any{"access_token": "at", "token_type": "Bearer", "expires_in": 3600, "id_token": idToken})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &idp.key.PublicKey, KeyID: "k1", Algorithm: "RS256", Use: "sig"}}})
	})
	idp.Server = httptest.NewServer(mux)
	t.Cleanup(idp.Close)
	return idp
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	assert.NoError(t, json.NewEncoder(w).Encode(v))
}

// newSSOArgoServer serves the Argo Server's /oauth2 endpoints against idp.
func newSSOArgoServer(t *testing.T, idp *fakeIDP) (*httptest.Server, sso.Interface) {
	t.Helper()
	var ssoIf sso.Interface
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth2/redirect", func(w http.ResponseWriter, r *http.Request) { ssoIf.HandleRedirect(w, r) })
	mux.HandleFunc("/oauth2/callback", func(w http.ResponseWriter, r *http.Request) { ssoIf.HandleCallback(w, r) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "UI "+r.URL.Path) })
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	secrets := fake.NewClientset(&apiv1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: "argo", Name: "argo-server-sso"},
		Data:       map[string][]byte{"client-id": []byte(testClientID), "client-secret": []byte("secret")},
	}).CoreV1().Secrets("argo")
	var err error
	ssoIf, err = sso.New(logging.TestContext(t.Context()), sso.Config{
		Issuer:       idp.URL,
		ClientID:     apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "argo-server-sso"}, Key: "client-id"},
		ClientSecret: apiv1.SecretKeySelector{LocalObjectReference: apiv1.LocalObjectReference{Name: "argo-server-sso"}, Key: "client-secret"},
		RedirectURL:  server.URL + "/oauth2/callback",
	}, secrets, "/", false)
	require.NoError(t, err)
	return server, ssoIf
}

// browser follows redirects and keeps cookies, like a real browser.
type browser struct {
	*http.Client
}

func newBrowser(t *testing.T) *browser {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &browser{&http.Client{Jar: jar}}
}

// page is where a browser visit ended.
type page struct {
	URL    *url.URL
	Status int
	Body   string
}

func (b *browser) visit(ctx context.Context, u string) (*page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	return &page{URL: resp.Request.URL, Status: resp.StatusCode, Body: string(body)}, err
}

func TestSSOBrowserLoginEndToEnd(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	idp := newFakeIDP(t)
	argoServer, ssoIf := newSSOArgoServer(t, idp)

	b := newBrowser(t)
	done := make(chan struct{})
	t.Cleanup(func() { <-done })
	stubBrowser(t, func(ctx context.Context, loginURL string) error {
		go func() {
			defer close(done)
			p, err := b.visit(context.WithoutCancel(ctx), loginURL)
			if assert.NoError(t, err) {
				assert.Equal(t, http.StatusOK, p.Status)
				assert.Equal(t, "127.0.0.1", p.URL.Hostname(), "the browser should end on the CLI's loopback page")
				assert.NotEqual(t, strings.TrimPrefix(argoServer.URL, "http://"), p.URL.Host, "the browser should leave the Argo Server")
				assert.Contains(t, p.Body, "You're logged in")
			}
		}()
		return nil
	})

	payload, err := browserLogin(ctx, argoServer.URL, true, 30*time.Second, io.Discard)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(payload.Token, sso.Prefix))
	assert.WithinDuration(t, time.Now().Add(10*time.Hour), payload.Expiry, time.Minute)

	claims, err := ssoIf.Authorize(payload.Token)
	require.NoError(t, err, "the Argo Server should accept the token the CLI received")
	assert.Equal(t, "alice@example.com", claims.Email)
	assert.Equal(t, []string{"data-eng"}, claims.Groups)
}

func TestSSOPasteLoginEndToEnd(t *testing.T) {
	ctx := logging.TestContext(t.Context())
	idp := newFakeIDP(t)
	argoServer, ssoIf := newSSOArgoServer(t, idp)

	b := newBrowser(t)
	stdin, typist := io.Pipe()
	stubBrowser(t, func(ctx context.Context, loginURL string) error {
		p, err := b.visit(ctx, loginURL)
		require.NoError(t, err)
		assert.Equal(t, "/userinfo", p.URL.Path, "the browser should end on the User Info page")
		assert.Equal(t, "UI /userinfo", p.Body)
		// The User Info page shows the authorization cookie as an export line.
		u, err := url.Parse(argoServer.URL)
		require.NoError(t, err)
		var token string
		for _, c := range b.Jar.Cookies(u) {
			if c.Name == "authorization" {
				token = c.Value
			}
		}
		require.NotEmpty(t, token)
		go func() {
			_, _ = io.WriteString(typist, "export ARGO_TOKEN='"+token+"'\n")
			_ = typist.Close()
		}()
		return nil
	})

	payload, err := pasteLogin(ctx, argoServer.URL, true, stdin, io.Discard)
	require.NoError(t, err)
	claims, err := ssoIf.Authorize(payload.Token)
	require.NoError(t, err, "the Argo Server should accept the pasted token")
	assert.Equal(t, "alice@example.com", claims.Email)
}

func TestSSOUILoginUnchanged(t *testing.T) {
	idp := newFakeIDP(t)
	argoServer, ssoIf := newSSOArgoServer(t, idp)
	b := newBrowser(t)

	p, err := b.visit(t.Context(), argoServer.URL+"/oauth2/redirect?redirect=/workflows")
	require.NoError(t, err)
	assert.Equal(t, argoServer.URL+"/workflows", p.URL.String())
	assert.Equal(t, "UI /workflows", p.Body)
	u, err := url.Parse(argoServer.URL)
	require.NoError(t, err)
	var found bool
	for _, c := range b.Jar.Cookies(u) {
		if c.Name == "authorization" {
			found = true
			_, err := ssoIf.Authorize(c.Value)
			require.NoError(t, err)
		}
	}
	assert.True(t, found, "the UI login should still set the authorization cookie")
}

func TestSSOLoginRefusesNonLoopbackRedirect(t *testing.T) {
	idp := newFakeIDP(t)
	argoServer, _ := newSSOArgoServer(t, idp)
	session, err := clilogin.NewSession()
	require.NoError(t, err)
	params, err := session.Params("http://127.0.0.1:1/callback")
	require.NoError(t, err)

	for name, redirect := range map[string]string{
		"remote host": "http://attacker.example.com:8085/callback",
		"localhost":   "http://localhost:8085/callback",
		"https":       "https://127.0.0.1:8085/callback",
	} {
		t.Run(name, func(t *testing.T) {
			params.Set(clilogin.ParamRedirect, redirect)
			b := newBrowser(t)
			b.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				t.Errorf("the Argo Server must not redirect anywhere, but redirected to %s", req.URL)
				return http.ErrUseLastResponse
			}
			p, err := b.visit(t.Context(), argoServer.URL+"/oauth2/redirect?"+params.Encode())
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, p.Status)
		})
	}
}
