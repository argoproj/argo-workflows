package impersonate

import (
	"fmt"
	"strings"
)

type Config struct {
	Enabled       bool   `json:"enabled,omitempty"`
	UsernameClaim *Claim `json:"usernameClaim,omitempty"`
}

func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled
}

func (c *Config) GetUsernameClaim() Claim {
	if c.UsernameClaim != nil {
		return *c.UsernameClaim
	} else {
		// default to "email" when `sso.impersonate.usernameClaim` is nil/omitted
		return EmailClaim
	}
}

type Claim string

const (
	EmailClaim   Claim = "email"
	SubjectClaim Claim = "sub"
)

// UnmarshalJSON is a custom Unmarshal that overwrites json.Unmarshal for Claim
func (c *Claim) UnmarshalJSON(data []byte) error {
	str := strings.ToLower(strings.Trim(string(data), `"`))
	switch str {
	case "":
		// default to "email" when `sso.impersonate.usernameClaim` is ""
		*c = EmailClaim
	case "email":
		*c = EmailClaim
	case "sub":
		*c = SubjectClaim
	default:
		return fmt.Errorf("'%s' is not a valid `sso.impersonate.usernameClaim`, must be one of: {'email', 'sub'}", str)
	}
	return nil
}
