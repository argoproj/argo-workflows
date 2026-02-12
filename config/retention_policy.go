package config

// Workflow retention by number of workflows
type RetentionPolicy struct {
	// Completed is the number of completed Workflows to retain
	Completed int `json:"completed,omitempty"`
	// Failed is the number of failed Workflows to retain
	Failed int `json:"failed,omitempty"`
	// Errored is the number of errored Workflows to retain
	Errored int `json:"errored,omitempty"`
}
