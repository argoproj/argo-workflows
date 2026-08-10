package logout

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
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
		logoutURL = ""
	}
	return &Handler{baseHRef: baseHRef, redirectURL: redirectURL, secure: secure, logoutURL: logoutURL, clientID: clientID}
}

// ValidateRedirectURL validates the optional post-logout redirect URL supplied by an operator.
func ValidateRedirectURL(redirectURL string) error {
	if redirectURL != "" && !isAbsoluteHTTPURL(redirectURL) {
		return fmt.Errorf("--logout-redirect-url must be an absolute HTTP(S) URL: %q", redirectURL)
	}
	return nil
}

// ValidateEndSessionURL validates the optional end-session endpoint discovered from an OIDC provider.
func ValidateEndSessionURL(logoutURL string) error {
	if logoutURL != "" && !isAbsoluteHTTPURL(logoutURL) {
		return fmt.Errorf("oidc end-session endpoint must be an absolute HTTP(S) URL: %q", logoutURL)
	}
	return nil
}

func isAbsoluteHTTPURL(rawURL string) bool {
	parsedURL, err := url.ParseRequestURI(rawURL)
	return err == nil && parsedURL.Host != "" &&
		(strings.EqualFold(parsedURL.Scheme, "http") || strings.EqualFold(parsedURL.Scheme, "https"))
}

func constructLogoutURL(logoutURL, clientID, redirectURL string) string {
	if logoutURL == "" {
		return redirectURL
	}

	parsedURL, err := url.ParseRequestURI(logoutURL)
	if err != nil || parsedURL.Host == "" ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) {
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
