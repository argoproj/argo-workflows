package main

import (
	"fmt"
	"os"
	"time"

	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/cmd/server/apiserver"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo-server"
)

// NewRootCommand returns an new instance of the workflow-controller main entrypoint
func NewRootCommand() *cobra.Command {
	var (
		clientConfig clientcmd.ClientConfig
		logLevel     string // --loglevel
		authMode     string
		configMap    string
		port         int
	)

	var command = cobra.Command{
		Use: CLIName,
		RunE: func(c *cobra.Command, args []string) error {
			cli.SetLogLevel(logLevel)
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}
			config.Burst = 30
			config.QPS = 20.0

			namespace, _, err := clientConfig.Namespace()
			if err != nil {
				return err
			}

			kubeConfig := kubernetes.NewForConfigOrDie(config)
			wflientset := wfclientset.NewForConfigOrDie(config)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			opts := apiserver.ArgoServerOpts{
				Namespace:     namespace,
				WfClientSet:   wflientset,
				KubeClientset: kubeConfig,
				RestConfig:    config,
				AuthMode:      authMode,
			}
			err = opts.ValidateOpts()
			if err != nil {
				return err
			}
			apiServer := apiserver.NewArgoServer(opts)

			go apiServer.Run(ctx, port)

			// Wait forever
			select {}

		},
	}

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.AddCommand(cmdutil.NewVersionCmd(CLIName))
	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	command.Flags().StringVar(&authMode, "auth-mode", "server", "API server authentication mode. One of: client|server|hybrid")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	return &command
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
