package config

// WorkflowEvents configures how workflow events are emitted
type WorkflowEvents struct {
	// Enabled controls whether workflow events are emitted
	Enabled *bool `json:"enabled,omitempty"`
}

func (e WorkflowEvents) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}
