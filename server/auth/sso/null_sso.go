package sso

import (
	"fmt"
	"net/http"

	"github.com/argoproj/argo-workflows/v3/server/auth/types"
)

var NullSSO Interface = nullService{}

type nullService struct{}

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
