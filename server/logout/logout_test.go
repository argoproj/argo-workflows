package logout

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
		{name: "defaults to base href", baseHRef: "/argo/", wantRedirect: "/argo/"},
		{name: "uses configured redirect URL", baseHRef: "/argo/", redirectURL: "https://example.com/", secure: true, wantRedirect: "https://example.com/"},
		{name: "does not use the OIDC end-session endpoint with the default relative redirect", baseHRef: "/argo/", logoutURL: "https://idp.example.com/logout", clientID: "workflows", wantRedirect: "/argo/"},
		{name: "redirects through the OIDC end-session endpoint", baseHRef: "/argo/", redirectURL: "https://example.com/", logoutURL: "https://idp.example.com/logout?foo=bar", clientID: "workflows", wantRedirect: "https://idp.example.com/logout?client_id=workflows&foo=bar&post_logout_redirect_uri=https%3A%2F%2Fexample.com%2F"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, LogoutEndpoint, nil)

			NewHandler(tt.baseHRef, tt.redirectURL, tt.secure, tt.logoutURL, tt.clientID).ServeHTTP(recorder, request)

			response := recorder.Result()
			assert.Equal(t, http.StatusSeeOther, response.StatusCode)
			assert.Equal(t, tt.wantRedirect, response.Header.Get("Location"))
			cookies := response.Cookies()
			require.Len(t, cookies, 1)
			cookie := cookies[0]
			assert.Equal(t, "authorization", cookie.Name)
			assert.Empty(t, cookie.Value)
			assert.Equal(t, tt.baseHRef, cookie.Path)
			assert.Equal(t, -1, cookie.MaxAge)
			assert.Equal(t, tt.secure, cookie.Secure)
			assert.True(t, cookie.HttpOnly)
			assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		})
	}
}

func TestConstructLogoutURL(t *testing.T) {
	assert.Equal(t, "https://example.com/", constructLogoutURL("", "client", "https://example.com/"))
	assert.Equal(t, "https://example.com/logout?post_logout_redirect_uri=https%3A%2F%2Fapp.example.com%2F", constructLogoutURL("https://example.com/logout", "", "https://app.example.com/"))
	assert.Equal(t, "https://example.com/", constructLogoutURL("://bad", "client", "https://example.com/"))
	assert.Equal(t, "https://example.com/", constructLogoutURL("javascript://example.com/logout", "client", "https://example.com/"))
}

func TestValidateRedirectURL(t *testing.T) {
	assert.NoError(t, ValidateRedirectURL(""))
	assert.NoError(t, ValidateRedirectURL("https://example.com/signed-out"))
	assert.NoError(t, ValidateRedirectURL("HTTPS://example.com/signed-out"))
	require.Error(t, ValidateRedirectURL("/signed-out"))
	require.Error(t, ValidateRedirectURL("//example.com/signed-out"))
	require.Error(t, ValidateRedirectURL("javascript://example.com/signed-out"))
	require.Error(t, ValidateRedirectURL("https://example.com/signed-out#fragment"))
	require.Error(t, ValidateRedirectURL("https://:443/signed-out"))
}

func TestValidateEndSessionURL(t *testing.T) {
	assert.NoError(t, ValidateEndSessionURL(""))
	assert.NoError(t, ValidateEndSessionURL("https://idp.example.com/logout?client=workflows"))
	require.Error(t, ValidateEndSessionURL("/logout"))
	require.Error(t, ValidateEndSessionURL("https://:443/logout"))
}
