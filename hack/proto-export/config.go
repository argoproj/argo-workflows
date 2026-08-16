package main

// ArgoProto represents the structure of argo-proto.yaml
type ArgoProto struct {
	Version      string                      `yaml:"version"`
	Dependencies map[string]DependencyConfig `yaml:"dependencies"`
}

// DependencyConfig contains the configuration details for a dependency.
type DependencyConfig struct {
	Owner  string                  `yaml:"owner,omitempty"`
	Name   string                  `yaml:"name,omitempty"`
	Buf    *BufConfig              `yaml:"buf,omitempty"`
	Bundle map[string]BundleConfig `yaml:"bundle,omitempty"`
}

// BufConfig represents the buf-specific configuration.
type BufConfig struct {
	Ref string `yaml:"ref"`
}

// BundleConfig represents a bundled dependency configuration.
type BundleConfig struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
	Ref   string `yaml:"ref"`
}
