package config

// Disabled by default, can be enabled.
type Enableable struct {
	Enabled bool `json:"enabled,omitempty"`
}

func (e *Enableable) IsEnabled() bool {
	return e != nil && e.Enabled
}
