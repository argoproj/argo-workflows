package config

type Image struct {
	Command []string `json:"command"`
	Args    []string `json:"args,omitempty"`
}
