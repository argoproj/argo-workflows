package auth

import (
	"errors"
)

type Modes map[Mode]bool

type Mode string

const (
	Client Mode = "client"
	Server Mode = "server"
	SSO    Mode = "sso"
	Header Mode = "header"
)

func (m Modes) Add(value string) error {
	switch value {
	case "client", "server", "sso", "header":
		m[Mode(value)] = true
	case "hybrid":
		m[Client] = true
		m[Server] = true
	default:
		return errors.New("invalid mode")
	}
	return nil
}
