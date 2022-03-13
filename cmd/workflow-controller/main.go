package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/argoproj/pkg/cli"
	"github.com/argoproj/pkg/errors"
	kubecli "github.com/argoproj/pkg/kube/cli"
	"github.com/argoproj/pkg/stats"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeutil "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"

	// load authentication plugin for obtaining credentials from cloud providers.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	pprofutil "github.com/argoproj/argo-workflows/v3/util/pprof"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
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
		podCleanupWorkers        int    // --pod-cleanup-workers
		burst                    int
		qps                      float32
		namespaced               bool   // --namespaced
		managedNamespace         string // --managed-namespace
		executorPlugins          bool
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
			pprofutil.Init()

			config, err := clientConfig.ClientConfig()
			if err != nil {
				return err
			}
			version := argo.GetVersion()
			config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-controller", version.Version))
			config.Burst = burst
			config.QPS = qps

			namespace, _, err := clientConfig.Namespace()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			configs, kubernetesInterfaces, workflowInterfaces, metadataInterfaces, err := loadClusters(ctx, config, namespace)
			errors.CheckError(err)

			if !namespaced && managedNamespace != "" {
				log.Warn("ignoring --managed-namespace because --namespaced is false")
				managedNamespace = ""
			}
			if namespaced && managedNamespace == "" {
				managedNamespace = namespace
			}

			wfController, err := controller.NewWorkflowController(
				ctx,
				configs,
				kubernetesInterfaces,
				metadataInterfaces,
				workflowInterfaces,
				namespace,
				managedNamespace,
				executorImage,
				executorImagePullPolicy,
				containerRuntimeExecutor,
				configMap,
				executorPlugins,
			)
			errors.CheckError(err)

			go wfController.Run(ctx, workflowWorkers, workflowTTLWorkers, podCleanupWorkers)

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
	command.Flags().IntVar(&podCleanupWorkers, "pod-cleanup-workers", 4, "Number of pod cleanup workers")
	command.Flags().IntVar(&burst, "burst", 30, "Maximum burst for throttle.")
	command.Flags().Float32Var(&qps, "qps", 20.0, "Queries per second")
	command.Flags().BoolVar(&namespaced, "namespaced", false, "run workflow-controller as namespaced mode")
	command.Flags().StringVar(&managedNamespace, "managed-namespace", "", "namespace that workflow-controller watches, default to the installation namespace")
	command.Flags().BoolVar(&executorPlugins, "executor-plugins", false, "enable executor plugins")

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ARGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	if err := viper.BindPFlags(command.Flags()); err != nil {
		log.Fatal(err)
	}
	command.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			if err := command.Flags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				log.Fatal(err)
			}
		}
	})

	return &command
}

func loadClusters(ctx context.Context, config *restclient.Config, namespace string) (
	map[string]*restclient.Config,
	map[string]kubernetes.Interface,
	map[string]wfclientset.Interface,
	map[string]metadata.Interface,
	error,
) {
	configs := map[string]*restclient.Config{common.LocalCluster: config}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	secrets := client.CoreV1().Secrets(namespace)
	list, err := secrets.List(ctx, metav1.ListOptions{LabelSelector: common.LabelKeyCluster})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to list kubeconfig secrets: %w", err)
	}
	for _, secret := range list.Items {
		kc, err := clientcmd.Load(secret.Data["kubeconfig"])
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to load kubeconfig from secret %q: %w", secret.Name, err)
		}
		println(v1alpha1.MustMarshallJSON(kc))
		config, err := clientcmd.NewNonInteractiveClientConfig(*kc, kc.CurrentContext, &clientcmd.ConfigOverrides{}, clientcmd.NewDefaultClientConfigLoadingRules()).ClientConfig()
		if err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to create client config for secret %q: %w", secret.Name, err)
		}
		configs[secret.Labels[common.LabelKeyCluster]] = config
	}
	kubernetesInterfaces := map[string]kubernetes.Interface{}
	workflowInterfaces := map[string]wfclientset.Interface{}
	metadataInterfaces := map[string]metadata.Interface{}
	for cluster, config := range configs {
		logs.AddK8SLogTransportWrapper(config)
		metrics.AddMetricsTransportWrapper(config)
		kubernetesInterfaces[cluster], err = kubernetes.NewForConfig(config)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		workflowInterfaces[cluster], err = wfclientset.NewForConfig(config)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		metadataInterfaces[cluster], err = metadata.NewForConfig(config)
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}

	return configs, kubernetesInterfaces, workflowInterfaces, metadataInterfaces, nil
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
