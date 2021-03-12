package config

type RetentionPolicy struct {
	Completed int `json:"completed,omitempty"`
	Failed    int `json:"failed,omitempty"`
	Errored   int `json:"errored,omitempty"`
}
