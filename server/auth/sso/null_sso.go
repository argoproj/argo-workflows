package sso

import (
	"context"
	"fmt"
	"net/http"

	v1 "k8s.io/api/core/v1"

	"github.com/argoproj/argo/server/auth/jws"
)

var NullSSO Interface = nullService{}

type nullService struct{}

func (n nullService) GetServiceAccount([]string) (*v1.LocalObjectReference, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n nullService) Authorize(context.Context, string) (*jws.ClaimSet, error) {
	return nil, fmt.Errorf("not implemented")
}

func (n nullService) HandleRedirect(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (n nullService) HandleCallback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
