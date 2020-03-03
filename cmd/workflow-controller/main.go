package main

import (
	"context"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"

	wfclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo/util/cmd"
	"github.com/argoproj/argo/workflow/controller"
)

const (
	// CLIName is the name of the CLI
	CLIName = "workflow-controller"
)

// NewRootCommand returns an new instance of the workflow-controller main entrypoint
func NewRootCommand() *cobra.Command {
	var (
		clientConfig             clientcmd.ClientConfig
		configMap                string // --configmap
		executorImage            string // --executor-image
		executorImagePullPolicy  string // --executor-image-pull-policy
		containerRuntimeExecutor string
		logLevel                 string // --loglevel
		glogLevel                int    // --gloglevel
		workflowWorkers          int    // --workflow-workers
		podWorkers               int    // --pod-workers
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace
	)

	var command = cobra.Command{
		Use:   CLIName,
		Short: "workflow-controller is the controller to operate on workflows",
		RunE: func(c *cobra.Command, args []string) error {
			cli.SetLogLevel(logLevel)
			cli.SetGLogLevel(glogLevel)
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

			kubeclientset := kubernetes.NewForConfigOrDie(config)
			wfclientset := wfclientset.NewForConfigOrDie(config)

			if !namespaced && managedNamespace != "" {
				log.Warn("ignoring --managed-namespace because --namespaced is false")
				managedNamespace = ""
			}
			if namespaced && managedNamespace == "" {
				managedNamespace = namespace
			}

			// start a controller on instances of our custom resource
			wfController := controller.NewWorkflowController(config, kubeclientset, wfclientset, namespace, managedNamespace, executorImage, executorImagePullPolicy, containerRuntimeExecutor, configMap)
			err = wfController.ResyncConfig()
			if err != nil {
				return err
			}
			// TODO: following code will be updated in next major release to remove configmap
			// setting for namespace installation mode.
			if len(wfController.Config.Namespace) > 0 {
				_, _ = fmt.Fprintf(os.Stderr, "\n------------------------    WARNING    ------------------------\n")
				_, _ = fmt.Fprintf(os.Stderr, "Namespaced installation with configmap setting is deprecated, \n")
				_, _ = fmt.Fprintf(os.Stderr, "it will be removed in next major release. Instead please add \n")
				_, _ = fmt.Fprintf(os.Stderr, "\"--namespaced\" to workflow-controller start args.\n")
				_, _ = fmt.Fprintf(os.Stderr, "-----------------------------------------------------------------\n\n")
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go wfController.Run(ctx, workflowWorkers, podWorkers)
			go wfController.MetricsServer(ctx)
			go wfController.TelemetryServer(ctx)
			go wfController.RunTTLController(ctx)
			go wfController.RunCronController(ctx)

			// Wait forever
			select {}
		},
	}

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.AddCommand(cmdutil.NewVersionCmd(CLIName))
	command.Flags().StringVar(&configMap, "configmap", "workflow-controller-configmap", "Name of K8s configmap to retrieve workflow controller configuration")
	command.Flags().StringVar(&executorImage, "executor-image", "", "Executor image to use (overrides value in configmap)")
	command.Flags().StringVar(&executorImagePullPolicy, "executor-image-pull-policy", "", "Executor imagePullPolicy to use (overrides value in configmap)")
	command.Flags().StringVar(&containerRuntimeExecutor, "container-runtime-executor", "", "Container runtime executor to use (overrides value in configmap)")
	command.Flags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.Flags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.Flags().IntVar(&workflowWorkers, "workflow-workers", 8, "Number of workflow workers")
	command.Flags().IntVar(&podWorkers, "pod-workers", 8, "Number of pod workers")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run workflow-controller as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that workflow-controller watches, default to the installation namespace")
	return &command
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
