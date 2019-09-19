package main

import (
	"fmt"
	"github.com/argoproj/argo/cmd/server"
	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"
	"k8s.io/client-go/tools/clientcmd"
	"os"
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

			//kubeclientset := kubernetes.NewForConfigOrDie(config)
			wflientset := wfclientset.NewForConfigOrDie(config)

			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())

			var opts = server.ArgoServerOpts{Namespace: namespace, KubeClientset: wflientset}
			argoSvr := server.NewArgoServer(ctx, opts )
			defer cancel()
			go argoSvr.Run(ctx,8082)


			// Wait forever
			select {}

		},
	}

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.AddCommand(cmdutil.NewVersionCmd(CLIName))
	//command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	//command.Flags().StringVar(&executorImage, "executor-image", "", "Executor image to use (overrides value in configmap)")
	//command.Flags().StringVar(&executorImagePullPolicy, "executor-image-pull-policy", "", "Executor imagePullPolicy to use (overrides value in configmap)")
	command.Flags().StringVar(&logLevel, "loglevel", "debug", "Set the logging level. One of: debug|info|warn|error")
	//command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	//command.Flags().IntVar(&workflowWorkers, "workflow-workers", 8, "Number of workflow workers")
	//command.Flags().IntVar(&podWorkers, "pod-workers", 8, "Number of pod workers")
	return &command
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}