package api

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"

	"github.com/ghodss/yaml"
)

// Environment variables to look to when creating a new config
const (
	EnvClusterURL      = "ARGO_CLUSTER_URL"
	EnvClusterUsername = "ARGO_CLUSTER_USERNAME"
	EnvClusterPassword = "ARGO_CLUSTER_PASSWORD"
)

// ClusterConfig are settings to use when creating a API client
type ClusterConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Insecure *bool  `json:"insecure,omitempty"`
}

// NewClusterConfig returns a new cluster config with settings obtained from
// environment variables, or looking to the default .argo/default location (best effort).
func NewClusterConfig() ClusterConfig {
	config := ClusterConfig{}
	usr, err := user.Current()
	if err == nil {
		argoConfigFile := path.Join(usr.HomeDir, ArgoDir, DefaultConfigName)
		fileInfo, err := os.Stat(argoConfigFile)
		if err == nil && fileInfo.Mode().IsRegular() {
			_ = config.FromFile(argoConfigFile)
		}
	}
	config.FromEnv()
	return config
}

// FromFile sets config settings from a config path
func (c *ClusterConfig) FromFile(path string) error {
	cfg, err := parseConfigFile(path)
	if err != nil {
		return err
	}
	c.FromConfig(*cfg)
	return nil
}

// FromEnv sets config settings from environment variables
func (c *ClusterConfig) FromEnv() {
	envClusterURL := os.Getenv(EnvClusterURL)
	if envClusterURL != "" {
		c.URL = envClusterURL
	}
	envUsername := os.Getenv(EnvClusterUsername)
	if envUsername != "" {
		c.Username = envUsername
	}
	envPassword := os.Getenv(EnvClusterPassword)
	if envPassword != "" {
		c.Password = envPassword
	}
}

// FromConfig sets config settings from another config
func (c *ClusterConfig) FromConfig(from ClusterConfig) {
	if from.URL != "" {
		c.URL = from.URL
	}
	if from.Username != "" {
		c.Username = from.Username
	}
	if from.Password != "" {
		c.Password = from.Password
	}
	if from.Insecure != nil {
		c.Insecure = from.Insecure
	}
}

// parseConfigFile parses the file at the given path and returns the config
func parseConfigFile(configPath string) (*ClusterConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("Can't read from file: %s: %s", configPath, err)
	}
	var config ClusterConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("%s had unexpected format", configPath)
	}
	return &config, nil
}

// WriteConfigFile writes the cluster config to a file
func (cfg *ClusterConfig) WriteConfigFile(configPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	err = os.MkdirAll(path.Dir(configPath), os.ModePerm)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(configPath, data, os.ModePerm)
}
