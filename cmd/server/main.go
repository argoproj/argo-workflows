package main

import (
	"fmt"
	"github.com/argoproj/argo/cmd/server/apiserver"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"strconv"
	"time"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo-api-server"
)

// NewRootCommand returns an new instance of the workflow-controller main entrypoint
func NewRootCommand() *cobra.Command {
	var (
		clientConfig            clientcmd.ClientConfig
		logLevel                string // --loglevel
		enableClientAuth		string
		configMap				string
		port					int
	)

	var command = cobra.Command{
		Use:   CLIName,
		Short: "Argo api server",
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

			if err != nil {
				return err
			}
			ctx, cancel := context.WithCancel(context.Background())
			var clientAuth bool
			clientAuth, err =strconv.ParseBool( enableClientAuth)
			var opts = apiserver.ArgoServerOpts{Namespace: namespace, WfClientSet: wflientset,KubeClientset: kubeConfig, EnableClientAuth: clientAuth}
			argoSvr := apiserver.NewArgoServer(ctx, opts )
			defer cancel()
			go argoSvr.Run(ctx,port)

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