package config

type NodeEvents struct {
	Enabled   *bool `json:"enabled,omitempty"`
	SendAsPod bool  `json:"sendAsPod,omitempty"`
}

func (e NodeEvents) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}
