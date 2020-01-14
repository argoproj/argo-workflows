package commands

import (
	"time"

	"github.com/argoproj/pkg/cli"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/server/apiserver"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
)

func NewServerCommand() *cobra.Command {
	var (
		logLevel  string // --loglevel
		authMode  string
		configMap string
		port      int
	)

	var command = cobra.Command{
		Use:   "server",
		Short: "Start the server",
		RunE: func(c *cobra.Command, args []string) error {
			cli.SetLogLevel(logLevel)
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := client.Config.ClientConfig()
			if err != nil {
				return err
			}
			config.Burst = 30
			config.QPS = 20.0

			namespace, _, err := client.Config.Namespace()
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
			apiserver.NewArgoServer(opts).Run(ctx, port)
			return nil
		},
	}

	command.Flags().IntVarP(&port, "port", "p", 2746, "Port to listen on")
	command.Flags().StringVar(&authMode, "auth-mode", "server", "API server authentication mode. One of: client|server|hybrid")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	return &command
}
