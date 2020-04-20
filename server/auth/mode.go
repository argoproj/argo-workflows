package auth

import "errors"

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