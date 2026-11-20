package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/argoproj/argo-workflows/v4/cmd/argo/commands/client"
	infopkg "github.com/argoproj/argo-workflows/v4/pkg/apiclient/info"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso"
	"github.com/argoproj/argo-workflows/v4/server/auth/sso/clilogin"
	"github.com/argoproj/argo-workflows/v4/util/logging"
)

// fakeServerBrowser stands in for the browser and the Argo Server: it runs the
// server's half of the handoff and follows the redirect to the CLI.
func fakeServerBrowser(t *testing.T, tamper func(url.Values)) func(context.Context, string) error {
	t.Helper()
	return func(ctx context.Context, loginURL string) error {
		u, err := url.Parse(loginURL)
		require.NoError(t, err)
		assert.Equal(t, "/oauth2/redirect", u.Path)
		request, err := clilogin.ParseRequest(u.Query())
		require.NoError(t, err)
		require.NotNil(t, request)
		location, err := request.Seal(clilogin.Payload{Token: "Bearer v2:abc", Expiry: time.Now().Add(time.Hour)})
		require.NoError(t, err)
		// The browser outlives browserLogin, which returns as soon as it has the
		// token, so don't cancel its requests with the login's context, and wait
		// for it before the test ends.
		ctx = context.WithoutCancel(ctx)
		done := make(chan struct{})
		t.Cleanup(func() { <-done })
		go func() {
			defer close(done)
			if tamper != nil {
				bad, _ := url.Parse(location)
				q := bad.Query()
				tamper(q)
				bad.RawQuery = q.Encode()
				resp, err := get(ctx, bad.String())
				if assert.NoError(t, err) {
					_ = resp.Body.Close()
					assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
				}
			}
			resp, err := get(ctx, location)
			if assert.NoError(t, err) {
				_ = resp.Body.Close()
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		}()
		return nil
	}
}

func stubBrowser(t *testing.T, f func(context.Context, string) error) {
	t.Helper()
	original := openBrowser
	openBrowser = f
	t.Cleanup(func() { openBrowser = original })
}

func TestBrowserLogin(t *testing.T) {
	stubBrowser(t, fakeServerBrowser(t, nil))
	var stderr bytes.Buffer
	payload, err := browserLogin(logging.TestContext(t.Context()), "https://argo.example.com", true, 10*time.Second, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "Bearer v2:abc", payload.Token)
	assert.False(t, payload.Expiry.IsZero())
	assert.Contains(t, stderr.String(), "https://argo.example.com/oauth2/redirect?")
}

func TestBrowserLoginIgnoresForgedCallback(t *testing.T) {
	stubBrowser(t, fakeServerBrowser(t, func(q url.Values) { q.Set(clilogin.ResponseParamState, "forged") }))
	var stderr bytes.Buffer
	payload, err := browserLogin(logging.TestContext(t.Context()), "https://argo.example.com", true, 10*time.Second, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "Bearer v2:abc", payload.Token)
	assert.Contains(t, stderr.String(), "Ignored a login callback")
}

func TestBrowserLoginTimeout(t *testing.T) {
	stubBrowser(t, func(context.Context, string) error { return nil })
	_, err := browserLogin(logging.TestContext(t.Context()), "https://argo.example.com", true, 50*time.Millisecond, &bytes.Buffer{})
	require.ErrorContains(t, err, "--paste")
}

func TestPasteLogin(t *testing.T) {
	original := client.ArgoServerOpts
	t.Cleanup(func() { client.ArgoServerOpts = original })
	client.ArgoServerOpts.Path = "/argo/"
	stubBrowser(t, func(context.Context, string) error {
		t.Fatal("the browser must not open with --no-browser")
		return nil
	})

	var stderr bytes.Buffer
	stdin := strings.NewReader("export ARGO_TOKEN='Bearer v2:abc'\n")
	payload, err := pasteLogin(logging.TestContext(t.Context()), "https://argo.example.com/argo", false, stdin, &stderr)
	require.NoError(t, err)
	assert.Equal(t, "Bearer v2:abc", payload.Token)
	assert.True(t, payload.Expiry.IsZero())
	assert.Contains(t, stderr.String(), "https://argo.example.com/argo/oauth2/redirect?redirect=%2Fargo%2Fuserinfo")
}

func TestPasteLoginEmpty(t *testing.T) {
	_, err := pasteLogin(logging.TestContext(t.Context()), "https://argo.example.com", false, strings.NewReader(""), &bytes.Buffer{})
	require.Error(t, err)
}

func TestNormalizeToken(t *testing.T) {
	for input, want := range map[string]string{
		"Bearer v2:abc":                       "Bearer v2:abc",
		"v2:abc":                              "Bearer v2:abc",
		"  Bearer v2:abc \n":                  "Bearer v2:abc",
		"'Bearer v2:abc'":                     "Bearer v2:abc",
		"ARGO_TOKEN=\"Bearer v2:abc\"":        "Bearer v2:abc",
		"export ARGO_TOKEN='Bearer v2:abc'":   "Bearer v2:abc",
		"export ARGO_TOKEN='Bearer v2:abc'\r": "Bearer v2:abc",
	} {
		t.Run(input, func(t *testing.T) {
			got, err := normalizeToken(input)
			require.NoError(t, err)
			assert.Equal(t, want, got)
		})
	}
	for _, input := range []string{"", "   ", "''", "not a token", "bad", "Bearer bad", "Bearer eyJhbGciOi.k8s.token", "Basic YWRtaW46cGFzcw=="} {
		t.Run("rejects "+input, func(t *testing.T) {
			_, err := normalizeToken(input)
			require.Error(t, err)
		})
	}
}

func get(ctx context.Context, u string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func TestCheckTokenRejectsNonSSOTokens(t *testing.T) {
	// A non-SSO token would pass as the server's identity on a server that
	// also allows the `server` auth mode, so it's rejected before any request.
	for _, token := range []string{"Bearer bad", "Bearer eyJhbGciOi.k8s.token", "Basic YWRtaW46cGFzcw=="} {
		_, err := checkToken(logging.TestContext(t.Context()), token)
		require.ErrorContains(t, err, "isn't an Argo Server SSO token")
	}
}

func TestLoginTimeoutLimitedToLoginState(t *testing.T) {
	original := client.ArgoServerOpts
	t.Cleanup(func() { client.ArgoServerOpts = original })
	client.ArgoServerOpts.URL = "argo.example.com:443"
	stubBrowser(t, func(context.Context, string) error {
		t.Fatal("the login must not start with an invalid --timeout")
		return nil
	})
	for _, timeout := range []string{"0s", "-1m", "3m1s", "10m"} {
		t.Run(timeout, func(t *testing.T) {
			cmd := NewLoginCommand()
			cmd.SetArgs([]string{"--timeout", timeout})
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			require.ErrorContains(t, cmd.ExecuteContext(logging.TestContext(t.Context())), "--timeout")
		})
	}
	assert.Equal(t, clilogin.LoginStateLifetime.String(), NewLoginCommand().Flags().Lookup("timeout").DefValue)
}

func TestSSOUser(t *testing.T) {
	t.Run("server-only Argo Server", func(t *testing.T) {
		// Without SSO, the Argo Server answers `Bearer v2:bad` in `server` mode
		// with an empty response, so the token must not be saved.
		_, err := ssoUser(&infopkg.GetUserInfoResponse{})
		require.ErrorContains(t, err, "didn't accept the token as an SSO login")
	})
	t.Run("client mode token", func(t *testing.T) {
		_, err := ssoUser(&infopkg.GetUserInfoResponse{Issuer: "https://kubernetes.default.svc", Subject: "system:serviceaccount:argo:default"})
		require.ErrorContains(t, err, "didn't accept the token as an SSO login")
	})
	t.Run("SSO token", func(t *testing.T) {
		user, err := ssoUser(&infopkg.GetUserInfoResponse{Issuer: sso.Issuer, Subject: "alice", Email: "alice@example.com"})
		require.NoError(t, err)
		assert.Equal(t, "alice@example.com", user)
	})
	t.Run("SSO token without names", func(t *testing.T) {
		user, err := ssoUser(&infopkg.GetUserInfoResponse{Issuer: sso.Issuer})
		require.NoError(t, err)
		assert.Equal(t, "unknown user", user)
	})
}
