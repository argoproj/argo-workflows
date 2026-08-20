package logout

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	authcookie "github.com/argoproj/argo-workflows/v4/server/auth/cookie"
)

// LogoutEndpoint is the HTTP endpoint used to clear the local Argo Workflows session and, when configured, initiate OIDC logout.
const LogoutEndpoint = "/auth/logout"

type Handler struct {
	cookiePaths []string
	redirectURL string
	secure      bool
}

// NewHandler creates a handler that clears the Argo Workflows authorization cookie and redirects the user.
// If the provider end-session URL is invalid, the handler falls back to the local redirect and returns the validation error.
func NewHandler(baseHRef, redirectURL string, secure bool, logoutURL, clientID string) (*Handler, error) {
	baseHRef = normalizeBaseHRef(baseHRef)
	cookiePaths := []string{baseHRef}
	if legacyCookiePath := strings.TrimSuffix(baseHRef, "/"); legacyCookiePath != "" {
		cookiePaths = append(cookiePaths, legacyCookiePath)
	}
	if redirectURL == "" {
		redirectURL = baseHRef
		logoutURL = ""
	}
	finalRedirectURL, err := constructLogoutURL(logoutURL, clientID, redirectURL)
	return &Handler{cookiePaths: cookiePaths, redirectURL: finalRedirectURL, secure: secure}, err
}

func normalizeBaseHRef(baseHRef string) string {
	trimmed := strings.Trim(baseHRef, "/")
	if trimmed == "" {
		return "/"
	}
	return "/" + trimmed + "/"
}

// ValidateRedirectURL validates the optional post-logout redirect URL supplied by an operator.
func ValidateRedirectURL(redirectURL string) error {
	if redirectURL != "" && !isAbsoluteHTTPURL(redirectURL) {
		return fmt.Errorf("logout redirect URL must be an absolute HTTP(S) URL without a fragment: %q", redirectURL)
	}
	return nil
}

func isAbsoluteHTTPURL(rawURL string) bool {
	_, ok := parseAbsoluteHTTPURL(rawURL)
	return ok
}

func parseAbsoluteHTTPURL(rawURL string) (*url.URL, bool) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Hostname() == "" || parsedURL.Fragment != "" ||
		(!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https")) {
		return nil, false
	}
	return parsedURL, true
}

func constructLogoutURL(logoutURL, clientID, redirectURL string) (string, error) {
	if logoutURL == "" {
		return redirectURL, nil
	}

	parsedURL, ok := parseAbsoluteHTTPURL(logoutURL)
	if !ok {
		return redirectURL, fmt.Errorf("oidc end-session endpoint must be an absolute HTTP(S) URL without a fragment: %q", logoutURL)
	}

	query := parsedURL.Query()
	if clientID != "" {
		query.Set("client_id", clientID)
	}
	if redirectURL != "" {
		query.Set("post_logout_redirect_uri", redirectURL)
	}
	parsedURL.RawQuery = query.Encode()
	return parsedURL.String(), nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, cookiePath := range h.cookiePaths {
		authcookie.ClearAuthCookie(w, cookiePath, h.secure)
	}
	http.Redirect(w, r, h.redirectURL, http.StatusSeeOther)
}
