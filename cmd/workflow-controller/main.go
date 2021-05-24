package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/argoproj/pkg/cli"
	"github.com/argoproj/pkg/errors"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/controller"
	"github.com/argoproj/argo-workflows/v3/workflow/metrics"
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
		logFormat                string // --log-format
		workflowWorkers          int    // --workflow-workers
		workflowTTLWorkers       int    // --workflow-ttl-workers
		podWorkers               int    // --pod-workers
		podCleanupWorkers        int    // --pod-cleanup-workers
		burst                    int
		qps                      float32
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace

	)

	command := cobra.Command{
		Use:   CLIName,
		Short: "workflow-controller is the controller to operate on workflows",
		RunE: func(c *cobra.Command, args []string) error {
			defer runtimeutil.HandleCrash(runtimeutil.PanicHandlers...)

			cli.SetLogLevel(logLevel)
			cmdutil.SetGLogLevel(glogLevel)
			cmdutil.SetLogFormatter(logFormat)
			stats.RegisterStackDumper()
			stats.StartStatsTicker(5 * time.Minute)

			config, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}
			version := argo.GetVersion()
			config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-controller", version.Version))
			config.Burst = burst
			config.QPS = qps

			logs.AddK8SLogTransportWrapper(config)
			metrics.AddMetricsTransportWrapper(config)

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
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wfController, err := controller.NewWorkflowController(ctx, config, kubeclientset, wfclientset, namespace, managedNamespace, executorImage, executorImagePullPolicy, containerRuntimeExecutor, configMap)
			errors.CheckError(err)

			go wfController.Run(ctx, workflowWorkers, workflowTTLWorkers, podWorkers, podCleanupWorkers)

			http.HandleFunc("/healthz", wfController.Healthz)

			go func() {
				log.Println(http.ListenAndServe(":6060", nil))
			}()

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
	command.Flags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")
	command.Flags().IntVar(&workflowWorkers, "workflow-workers", 32, "Number of workflow workers")
	command.Flags().IntVar(&workflowTTLWorkers, "workflow-ttl-workers", 4, "Number of workflow TTL workers")
	command.Flags().IntVar(&podWorkers, "pod-workers", 32, "Number of pod workers")
	command.Flags().IntVar(&podCleanupWorkers, "pod-cleanup-workers", 4, "Number of pod cleanup workers")
	command.Flags().IntVar(&burst, "burst", 30, "Maximum burst for throttle.")
	command.Flags().Float32Var(&qps, "qps", 20.0, "Queries per second")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run workflow-controller as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that workflow-controller watches, default to the installation namespace")
	return &command
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000Z",
		FullTimestamp:   true,
	})
}

func main() {
	if err := NewRootCommand().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
