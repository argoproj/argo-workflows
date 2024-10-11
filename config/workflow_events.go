package config

type WorkflowEvents struct {
	Enabled   *bool `json:"enabled,omitempty"`
}

func (e WorkflowEvents) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}
