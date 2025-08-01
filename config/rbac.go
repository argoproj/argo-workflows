package config

type RBACConfig struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (c *RBACConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}
