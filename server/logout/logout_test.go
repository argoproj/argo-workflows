package logout

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authcookie "github.com/argoproj/argo-workflows/v4/server/auth/cookie"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoutHandler(t *testing.T) {
	for _, tt := range []struct {
		name         string
		baseHRef     string
		redirectURL  string
		secure       bool
		logoutURL    string
		clientID     string
		wantRedirect string
	}{
		{name: "defaults to normalized base href", baseHRef: "/argo", wantRedirect: "/argo/"},
		{name: "uses configured redirect URL", baseHRef: "/argo/", redirectURL: "https://example.com/", secure: true, wantRedirect: "https://example.com/"},
		{name: "does not use the OIDC end-session endpoint with the default relative redirect", baseHRef: "/argo/", logoutURL: "https://idp.example.com/logout", clientID: "workflows", wantRedirect: "/argo/"},
		{name: "redirects through the OIDC end-session endpoint", baseHRef: "/argo/", redirectURL: "https://example.com/", logoutURL: "https://idp.example.com/logout?foo=bar", clientID: "workflows", wantRedirect: "https://idp.example.com/logout?client_id=workflows&foo=bar&post_logout_redirect_uri=https%3A%2F%2Fexample.com%2F"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, LogoutEndpoint, nil)

			handler, err := NewHandler(tt.baseHRef, tt.redirectURL, tt.secure, tt.logoutURL, tt.clientID)
			require.NoError(t, err)
			handler.ServeHTTP(recorder, request)

			response := recorder.Result()
			assert.Equal(t, http.StatusSeeOther, response.StatusCode)
			assert.Equal(t, tt.wantRedirect, response.Header.Get("Location"))
			cookies := response.Cookies()
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, authcookie.AuthorizationCookieName, cookie.Name)
			assert.Empty(t, cookie.Value)
			assert.Equal(t, normalizeBaseHRef(tt.baseHRef), cookie.Path)
			assert.Equal(t, -1, cookie.MaxAge)
			assert.Equal(t, tt.secure, cookie.Secure)
			assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		})
	}
}

func TestConstructLogoutURL(t *testing.T) {
	redirect, err := constructLogoutURL("", "client", "https://example.com/")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/", redirect)
	redirect, err = constructLogoutURL("https://example.com/logout", "", "https://app.example.com/")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/logout?post_logout_redirect_uri=https%3A%2F%2Fapp.example.com%2F", redirect)
	redirect, err = constructLogoutURL("://bad", "client", "https://example.com/")
	require.Error(t, err)
	assert.Equal(t, "https://example.com/", redirect)
	redirect, err = constructLogoutURL("javascript://example.com/logout", "client", "https://example.com/")
	require.EqualError(t, err, "oidc end-session endpoint must be an absolute HTTP(S) URL without a fragment: \"javascript://example.com/logout\"")
	assert.Equal(t, "https://example.com/", redirect)
	redirect, err = constructLogoutURL("https://example.com/logout#/signed-out", "client", "https://example.com/")
	require.EqualError(t, err, "oidc end-session endpoint must be an absolute HTTP(S) URL without a fragment: \"https://example.com/logout#/signed-out\"")
	assert.Equal(t, "https://example.com/", redirect)
}

func TestValidateRedirectURL(t *testing.T) {
	assert.NoError(t, ValidateRedirectURL(""))
	assert.NoError(t, ValidateRedirectURL("https://example.com/signed-out"))
	assert.NoError(t, ValidateRedirectURL("HTTPS://example.com/signed-out"))
	require.EqualError(t, ValidateRedirectURL("/signed-out"), "logout redirect URL must be an absolute HTTP(S) URL without a fragment: \"/signed-out\"")
	require.Error(t, ValidateRedirectURL("//example.com/signed-out"))
	require.Error(t, ValidateRedirectURL("javascript://example.com/signed-out"))
	require.EqualError(t, ValidateRedirectURL("https://example.com/signed-out#fragment"), "logout redirect URL must be an absolute HTTP(S) URL without a fragment: \"https://example.com/signed-out#fragment\"")
	require.Error(t, ValidateRedirectURL("https://:443/signed-out"))
}
