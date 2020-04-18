package auth

import "errors"

type Mode string

const (
	Client Mode = "client"
	Hybrid Mode = "hybrid"
	Server Mode = "server"
	SSO    Mode = "sso"
)

func (m Mode) IsValid() error {
	switch m {
	case Client, Hybrid, Server, SSO:
		return nil
	}
	return errors.New("invalid mode")
}
