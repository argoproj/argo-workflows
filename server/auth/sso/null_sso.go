package sso

import (
	"fmt"
	"net/http"

	"github.com/argoproj/argo-workflows/v4/server/auth/types"
	"github.com/argoproj/argo-workflows/v4/server/logout"
)

var NullSSO Interface = nullService{}

type nullService struct {
	logoutRedirectURL string
}

// NewNullSSO creates a non-SSO service with the configured local logout redirect.
func NewNullSSO(logoutRedirectURL string) (Interface, error) {
	if err := logout.ValidateRedirectURL(logoutRedirectURL); err != nil {
		return nil, fmt.Errorf("invalid sso.logoutRedirectUrl: %w", err)
	}
	return nullService{logoutRedirectURL: logoutRedirectURL}, nil
}

func (n nullService) LogoutURL() string {
	return ""
}

func (n nullService) LogoutRedirectURL() string {
	return n.logoutRedirectURL
}

func (n nullService) ClientID() string {
	return ""
}

func (n nullService) IsRBACEnabled() bool {
	return false
}

func (n nullService) Authorize(string) (*types.Claims, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n nullService) HandleRedirect(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (n nullService) HandleCallback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
