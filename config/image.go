package config

// Image contains command and entrypoint configuration for container images
type Image struct {
	// Entrypoint overrides the container entrypoint
	Entrypoint []string `json:"entrypoint,omitempty"`
	// Cmd overrides the container command
	Cmd []string `json:"cmd,omitempty"`
}
