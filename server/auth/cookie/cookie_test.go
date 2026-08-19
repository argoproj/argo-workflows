package cookie

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthCookiesUseMatchingAttributes(t *testing.T) {
	const path = "/argo/"
	const value = "Bearer v2:token"
	expires := time.Now().Add(time.Hour)

	setRecorder := httptest.NewRecorder()
	SetAuthCookie(setRecorder, value, path, expires, true)
	setCookies := setRecorder.Result().Cookies()
	require.Len(t, setCookies, 1)
	setCookie := setCookies[0]

	clearRecorder := httptest.NewRecorder()
	ClearAuthCookie(clearRecorder, path, true)
	clearCookies := clearRecorder.Result().Cookies()
	require.Len(t, clearCookies, 1)
	clearCookie := clearCookies[0]

	assert.Equal(t, AuthorizationCookieName, setCookie.Name)
	assert.Equal(t, setCookie.Name, clearCookie.Name)
	assert.Equal(t, path, setCookie.Path)
	assert.Equal(t, setCookie.Path, clearCookie.Path)
	assert.Equal(t, setCookie.Domain, clearCookie.Domain)
	assert.Equal(t, http.SameSiteStrictMode, setCookie.SameSite)
	assert.Equal(t, setCookie.SameSite, clearCookie.SameSite)
	assert.True(t, setCookie.Secure)
	assert.Equal(t, setCookie.Secure, clearCookie.Secure)
	assert.Equal(t, setCookie.HttpOnly, clearCookie.HttpOnly)
	assert.Equal(t, -1, clearCookie.MaxAge)
}
