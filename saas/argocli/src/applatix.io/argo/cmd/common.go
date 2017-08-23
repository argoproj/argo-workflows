package cmd

import (
	"fmt"
	"log"
	"os/user"
	"path"

	"applatix.io/api"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

// Version information
var (
	Version     = "unknown"
	Revision    = "unknown"
	FullVersion = fmt.Sprintf("%s-%s", Version, Revision)
)

var (
	// Global CLI flags
	globalArgs globalFlags
	// API client instance
	apiClient *api.ArgoClient
)

type globalFlags struct {
	config        string            // --config
	trace         bool              // --trace
	clusterConfig api.ClusterConfig // --cluster, --username, --password
	noColor       bool              // --no-color
}

func init() {
	RootCmd.PersistentFlags().StringVar(&globalArgs.clusterConfig.URL, "cluster", "", "Argo cluster URL")
	RootCmd.PersistentFlags().StringVar(&globalArgs.clusterConfig.Username, "username", "", "Argo username")
	RootCmd.PersistentFlags().StringVar(&globalArgs.clusterConfig.Password, "password", "", "Argo password")
	RootCmd.PersistentFlags().StringVar(&globalArgs.config, "config", "", "Name or path to a Argo cluster config")
	RootCmd.PersistentFlags().BoolVar(&globalArgs.trace, "trace", false, "Log API requests")
	RootCmd.PersistentFlags().BoolVar(&globalArgs.noColor, "no-color", false, "Disable colorized output")
}

func initConfig() api.ClusterConfig {
	var config api.ClusterConfig
	if globalArgs.config == "" {
		// Instantiate a default config (will look to ~/.argo/default)
		config = api.NewClusterConfig()
	} else {
		// if --config was supplied, it can be either a path to a file, or the name of a config file under .argo
		if fileExists(globalArgs.config) {
			err := config.FromFile(globalArgs.config)
			checkFatal(err)
		} else {
			usr, err := user.Current()
			if err != nil {
				log.Fatalln("Could not determine the current user")
			}
			argoConfigFile := path.Join(usr.HomeDir, api.ArgoDir, globalArgs.config)
			if fileExists(argoConfigFile) {
				err := config.FromFile(argoConfigFile)
				checkFatal(err)
			} else {
				log.Fatalf("No config file found under '%s' or %s\n", globalArgs.config, argoConfigFile)
			}
		}
	}
	// Allow the --cluster, --username, --password flags to take highest precedence
	config.FromConfig(globalArgs.clusterConfig)

	if config.URL == "" || config.Username == "" || config.Password == "" {
		log.Fatalf("Could not determine cluster config from flags, environment, or config file\n")
	}
	return config
}

// initClient initializes the global API client for this invocation of the CLI
func initClient() {
	if apiClient != nil {
		return
	}
	config := initConfig()
	client := api.NewArgoClient(config)
	client.Trace = globalArgs.trace
	apiClient = &client
}
