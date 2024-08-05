package auth

import (
	"errors"
	"fmt"
	"strings"

	"github.com/argoproj/argo-workflows/v3/server/auth/sso"
)

type Modes map[Mode]bool

type Mode string

const (
	Client Mode = "client"
	Server Mode = "server"
	SSO    Mode = "sso"
)

func (m Modes) Add(value string) error {
	switch value {
	case "client", "server", "sso":
		m[Mode(value)] = true
	case "hybrid":
		m[Client] = true
		m[Server] = true
	default:
		return errors.New("invalid mode")
	}
	return nil
}

func (m Modes) GetMode(authorization string) (Mode, bool, error) {
	if authorization == "" {
		if m[Server] {
			return Server, true, nil
		}
		return "", false, errors.New("empty token")
	}

	if m[SSO] && strings.HasPrefix(authorization, sso.Prefix) {
		return SSO, true, nil
	}
	if m[Client] && (strings.HasPrefix(authorization, "Bearer ") || strings.HasPrefix(authorization, "Basic ")) {
		return Client, true, nil
	}
	return "", false, fmt.Errorf("token is missing a prefix. must be Bearer, Basic, or %s", sso.Prefix)
}
