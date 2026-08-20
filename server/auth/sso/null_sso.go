package sso

import (
	"fmt"
	"net/http"

	"github.com/argoproj/argo-workflows/v4/server/auth/types"
)

var NullSSO Interface = nullService{}

type nullService struct {
	logoutRedirectURL string
}

// NewNullSSO creates a non-SSO service with the configured local logout redirect.
func NewNullSSO(logoutRedirectURL string) Interface {
	return nullService{logoutRedirectURL: logoutRedirectURL}
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
