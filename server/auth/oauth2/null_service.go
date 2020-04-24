package oauth2

import (
	"context"
	"fmt"
	"net/http"

	wfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

var NullService Service = nullService{}

type nullService struct {
}

func (n nullService) Authorize(context.Context, string) (wfv1.User, error) {
	return wfv1.NullUser, fmt.Errorf("not implemented")
}

func (n nullService) HandleRedirect(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (n nullService) HandleCallback(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}
