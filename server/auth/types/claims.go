package types

import "gopkg.in/square/go-jose.v2/jwt"

type Claims struct {
	jwt.Claims
	Groups []string `json:"groups,omitempty"`
}
