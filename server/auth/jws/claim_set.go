package jws

type ClaimSet struct {
	Iss    string   `json:"iss"`
	Sub    string   `json:"sub,omitempty"`
	Groups []string `json:"groups,omitempty"`
}
