package sso

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc"
)

var NullSSO Interface = nullService{}

type nullService struct{}

func (n nullService) Authorize(context.Context, string) (*oidc.UserInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n nullService) HandleRedirect(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (n nullService) HandleCallback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
