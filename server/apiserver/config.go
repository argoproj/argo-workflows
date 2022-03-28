package apiserver

import (
	"github.com/argoproj/argo-workflows/v3/config"
	"github.com/argoproj/argo-workflows/v3/server/auth/sso"
)

type Config struct {
	config.Config
	// SSO in settings for single-sign on
	SSO sso.Config `json:"sso,omitempty"`
}
