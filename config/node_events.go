package config

// NodeEvents configures how node events are emitted
type NodeEvents struct {
	// Enabled controls whether node events are emitted
	Enabled *bool `json:"enabled,omitempty"`
	// SendAsPod emits events as if from the Pod instead of the Workflow with annotations linking the event to the Workflow
	SendAsPod bool `json:"sendAsPod,omitempty"`
}

func (e NodeEvents) IsEnabled() bool {
	return e.Enabled == nil || *e.Enabled
}
