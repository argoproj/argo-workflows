package rbac

type Config struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (c *Config) IsEnabled() bool {
	return c != nil && c.Enabled
}
