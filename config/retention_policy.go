package config

// Workflow retention by number of workflows
type RetentionPolicy struct {
	Completed int `json:"completed,omitempty"`
	Failed    int `json:"failed,omitempty"`
	Errored   int `json:"errored,omitempty"`
}
