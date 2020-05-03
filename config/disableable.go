package config

// Is enabled by default, can be disabled.
type Disableable struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func (e *Disableable) IsEnabled() bool {
	return e == nil || e.Enabled == nil || *e.Enabled
}
