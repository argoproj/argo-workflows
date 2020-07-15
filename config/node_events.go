package config

type NodeEvents struct {
	Enabled *bool `json:"enabled,omitempty"`
}

func (e NodeEvents) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}
