package auth

import (
	"errors"
	"strings"

	"github.com/argoproj/argo/server/auth/oauth2"
)

type Modes map[Mode]bool

type Mode string

const (
	Client Mode = "client"
	// DEPRECATED - use both Client + Server
	Hybrid Mode = "hybrid"
	Server Mode = "server"
	SSO    Mode = "sso"
)

func (m Modes) Add(value Mode) error {
	switch value {
	case Client, Server, SSO:
		m[value] = true
	case Hybrid:
		m[Client] = true
		m[Server] = true
	default:
		return errors.New("invalid mode")
	}
	return nil
}

func GetMode(authorisation string) (Mode, error) {
	if authorisation == "" {
		return Server, nil
	}
	if strings.HasPrefix(authorisation, oauth2.Prefix) {
		return SSO, nil
	}
	if strings.HasPrefix(authorisation, "Bearer ") || strings.HasPrefix(authorisation, "Basic ") {
		return Client, nil
	}
	if strings.HasPrefix(authorisation, oauth2.Prefix) {
		return SSO, nil
	}
	return "", errors.New("unrecognized token")
}
