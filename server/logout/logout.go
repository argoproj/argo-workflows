package logout

import (
	"net/http"
	"net/url"
	"time"
)

// LogoutEndpoint is the HTTP endpoint used to clear the local Argo Workflows session and, when configured, initiate OIDC logout.
const LogoutEndpoint = "/auth/logout"

type Handler struct {
	baseHRef    string
	redirectURL string
	secure      bool
	logoutURL   string
	clientID    string
}

// NewHandler creates a handler that clears the Argo Workflows authorization cookie and redirects the user.
func NewHandler(baseHRef, redirectURL string, secure bool, logoutURL, clientID string) *Handler {
	if redirectURL == "" {
		redirectURL = baseHRef
	}
	return &Handler{baseHRef: baseHRef, redirectURL: redirectURL, secure: secure, logoutURL: logoutURL, clientID: clientID}
}

func constructLogoutURL(logoutURL, clientID, redirectURL string) string {
	if logoutURL == "" {
		return redirectURL
	}

	parsedURL, err := url.Parse(logoutURL)
	if err != nil || parsedURL.Host == "" || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return redirectURL
	}

	query := parsedURL.Query()
	if clientID != "" {
		query.Set("client_id", clientID)
	}
	if redirectURL != "" {
		query.Set("post_logout_redirect_uri", redirectURL)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String()
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "authorization",
		Value:    "",
		Path:     h.baseHRef,
		Expires:  time.Unix(1, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, constructLogoutURL(h.logoutURL, h.clientID, h.redirectURL), http.StatusSeeOther)
}
