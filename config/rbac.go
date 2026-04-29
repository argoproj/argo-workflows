package config

// RBACConfig contains role-based access control configuration
type RBACConfig struct {
	// Enabled controls whether RBAC is enabled
	Enabled bool `json:"enabled,omitempty"`
}

func (c *RBACConfig) IsEnabled() bool {
	return c != nil && c.Enabled
}
