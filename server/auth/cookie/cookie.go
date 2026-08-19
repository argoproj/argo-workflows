package cookie

import (
	"net/http"
	"time"
)

const (
	// AuthorizationCookieName is the name of the cookie that carries an Argo authorization token.
	AuthorizationCookieName = "authorization"
	// AuthorizationMetadataKey is the lowercase gRPC metadata key for an Argo authorization token.
	AuthorizationMetadataKey = "authorization"
)

// SetAuthCookie writes the Argo authorization cookie.
func SetAuthCookie(w http.ResponseWriter, value, path string, expires time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthorizationCookieName,
		Value:    value,
		Path:     path,
		Expires:  expires,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}

// ClearAuthCookie expires the Argo authorization cookie.
func ClearAuthCookie(w http.ResponseWriter, path string, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     AuthorizationCookieName,
		Value:    "",
		Path:     path,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		SameSite: http.SameSiteStrictMode,
		Secure:   secure,
	})
}
