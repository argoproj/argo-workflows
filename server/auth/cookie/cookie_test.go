package cookie

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePath(t *testing.T) {
	for _, testCase := range []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "/"},
		{name: "root", path: "/", want: "/"},
		{name: "without slashes", path: "argo", want: "/argo/"},
		{name: "without trailing slash", path: "/argo", want: "/argo/"},
		{name: "with trailing slash", path: "/argo/", want: "/argo/"},
		{name: "nested path", path: "//argo/workflows//", want: "/argo/workflows/"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, NormalizePath(testCase.path))
		})
	}
}

func TestAuthCookiesUseExpectedAttributes(t *testing.T) {
	const path = "/argo/"
	const value = "Bearer v2:token"
	expires := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)

	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprintf("secure=%t", secure), func(t *testing.T) {
			setRecorder := httptest.NewRecorder()
			SetAuthCookie(setRecorder, value, path, expires, secure)
			setCookies := setRecorder.Result().Cookies()
			require.Len(t, setCookies, 1)
			setCookie := setCookies[0]

			clearRecorder := httptest.NewRecorder()
			ClearAuthCookie(clearRecorder, path, secure)
			clearCookies := clearRecorder.Result().Cookies()
			require.Len(t, clearCookies, 1)
			clearCookie := clearCookies[0]

			assert.Equal(t, AuthorizationCookieName, setCookie.Name)
			assert.Equal(t, value, setCookie.Value)
			assert.Equal(t, path, setCookie.Path)
			assert.Empty(t, setCookie.Domain)
			assert.True(t, expires.Equal(setCookie.Expires))
			assert.Equal(t, http.SameSiteStrictMode, setCookie.SameSite)
			assert.Equal(t, secure, setCookie.Secure)
			assert.False(t, setCookie.HttpOnly)

			assert.Equal(t, AuthorizationCookieName, clearCookie.Name)
			assert.Empty(t, clearCookie.Value)
			assert.Equal(t, path, clearCookie.Path)
			assert.Empty(t, clearCookie.Domain)
			assert.True(t, time.Unix(1, 0).Equal(clearCookie.Expires))
			assert.Equal(t, -1, clearCookie.MaxAge)
			assert.Equal(t, http.SameSiteStrictMode, clearCookie.SameSite)
			assert.Equal(t, secure, clearCookie.Secure)
			assert.False(t, clearCookie.HttpOnly)
		})
	}
}
