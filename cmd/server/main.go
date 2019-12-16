package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo/cmd/server/apiserver"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo-api-server"
)

// NewRootCommand returns an new instance of the workflow-controller main entrypoint
func NewRootCommand() *cobra.Command {
	var (
		clientConfig     clientcmd.ClientConfig
		logLevel         string // --loglevel
		enableClientAuth string
		configMap        string
		port             int
	)

	var command = cobra.Command{
		Use:   CLIName,
		Short: "argo-api-server is Argo's API server",
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

			clientAuth, err := strconv.ParseBool(enableClientAuth)
			if err != nil {
				return err
			}

			opts := apiserver.ArgoServerOpts{
				Namespace:        namespace,
				WfClientSet:      wflientset,
				KubeClientset:    kubeConfig,
				EnableClientAuth: clientAuth,
			}
			apiServer := apiserver.NewArgoServer(opts)

			go apiServer.Run(ctx, port)

			// Wait forever
			select {}

		},
	}

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.AddCommand(cmdutil.NewVersionCmd(CLIName))
	command.Flags().IntVar(&port, "port", 8080, "")
	command.Flags().StringVar(&enableClientAuth, "enableClientAuth", "false", "")
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().StringVar(&logLevel, "loglevel", "debug", "Set the logging level. One of: debug|info|warn|error")
	return &command
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
