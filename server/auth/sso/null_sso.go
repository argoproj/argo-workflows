package sso

import (
	"context"
	"fmt"
	"net/http"
)

var NullSSO Interface = nullService{}

type nullService struct {
}

func (n nullService) Authorize(context.Context, string) error {
	return fmt.Errorf("not implemented")
}

func (n nullService) HandleRedirect(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (n nullService) HandleCallback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
