package impersonate

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Config struct {
	Enabled       bool  `json:"enabled,omitempty"`
	UsernameClaim Claim `json:"usernameClaim,omitempty"`
}

// UnmarshalJSON is a custom Unmarshal that overwrites json.Unmarshal
func (c *Config) UnmarshalJSON(data []byte) error {
	type innerConfig Config
	inner := &innerConfig{
		// set the default `sso.impersonate.usernameClaim` as "email" when omitted
		UsernameClaim: EmailClaim,
	}
	err := json.Unmarshal(data, inner)
	if err != nil {
		return err
	}
	*c = Config(*inner)
	return nil
}

func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled
}

func (c *Config) GetUsernameClaim() Claim {
	return c.UsernameClaim
}

type Claim string

const (
	EmailClaim   Claim = "email"
	SubjectClaim Claim = "sub"
)

// UnmarshalJSON is a custom Unmarshal that overwrites json.Unmarshal
func (c *Claim) UnmarshalJSON(data []byte) error {
	str := strings.ToLower(strings.Trim(string(data), `"`))
	switch str {
	case "email", "":
		*c = EmailClaim
	case "sub":
		*c = SubjectClaim
	default:
		return fmt.Errorf("invalid `sso.impersonate.usernameClaim` '%s', must be one of: {'email', 'sub'}", str)
	}
	return nil
}
