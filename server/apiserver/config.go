package apiserver

import (
	"github.com/argoproj/argo/config"
	"github.com/argoproj/argo/server/auth/sso"
)

var emptyConfigFunc = func() interface{} { return &Config{} }

type Config struct {
	config.Config
	// SSO in settings for single-sign on
	SSO sso.Config `json:"sso,omitempty"`
}
