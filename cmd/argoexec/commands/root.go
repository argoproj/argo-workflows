package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

	"github.com/argoproj/pkg/cli"
	kubecli "github.com/argoproj/pkg/kube/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/argoproj/argo-workflows/v3/util/cmd"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/docker"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/emissary"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/k8sapi"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/kubelet"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/pns"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argoexec"
)

var (
	clientConfig clientcmd.ClientConfig
	logLevel     string // --loglevel
	glogLevel    int    // --gloglevel
	logFormat    string // --log-format
)

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	cmd.SetLogFormatter(logFormat)
	cli.SetLogLevel(logLevel)
	cmd.SetGLogLevel(glogLevel)
}

func NewRootCommand() *cobra.Command {
	command := cobra.Command{
		Use:   CLIName,
		Short: "argoexec is the executor sidecar to workflow containers",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewEmissaryCommand())
	command.AddCommand(NewInitCommand())
	command.AddCommand(NewResourceCommand())
	command.AddCommand(NewWaitCommand())
	command.AddCommand(NewDataCommand())
	command.AddCommand(cmd.NewVersionCmd(CLIName))

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")

	return &command
}

func initExecutor() *executor.WorkflowExecutor {
	version := argo.GetVersion()
	executorType := os.Getenv(common.EnvVarContainerRuntimeExecutor)
	log.WithFields(log.Fields{"version": version.Version, "executorType": executorType}).Info("Starting Workflow Executor")
	config, err := clientConfig.ClientConfig()
	checkErr(err)
	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor/%s", version.Version, executorType))

	logs.AddK8SLogTransportWrapper(config) // lets log all request as we should typically do < 5 per pod, so this is will show up problems

	namespace, _, err := clientConfig.Namespace()
	checkErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)

	restClient := clientset.RESTClient()

	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		log.Fatalf("Unable to determine pod name from environment variable %s", common.EnvVarPodName)
	}

	tmpl := &wfv1.Template{}
	checkErr(json.Unmarshal([]byte(os.Getenv(common.EnvVarTemplate)), tmpl))

	includeScriptOutput := os.Getenv(common.EnvVarIncludeScriptOutput) == "true"
	deadline, err := time.Parse(time.RFC3339, os.Getenv(common.EnvVarDeadline))
	checkErr(err)

	var cre executor.ContainerRuntimeExecutor
	switch executorType {
	case common.ContainerRuntimeExecutorK8sAPI:
		cre = k8sapi.NewK8sAPIExecutor(clientset, config, podName, namespace)
	case common.ContainerRuntimeExecutorKubelet:
		cre, err = kubelet.NewKubeletExecutor(namespace, podName)
	case common.ContainerRuntimeExecutorPNS:
		cre, err = pns.NewPNSExecutor(clientset, podName, namespace)
	case common.ContainerRuntimeExecutorEmissary:
		cre, err = emissary.New()
	default:
		cre, err = docker.NewDockerExecutor(namespace, podName)
	}
	checkErr(err)

	wfExecutor := executor.NewExecutor(clientset, restClient, podName, namespace, cre, *tmpl, includeScriptOutput, deadline)
	log.
		WithField("version", version.String()).
		WithField("namespace", namespace).
		WithField("podName", podName).
		WithField("template", wfv1.MustMarshallJSON(&wfExecutor.Template)).
		WithField("includeScriptOutput", includeScriptOutput).
		WithField("deadline", deadline).
		Info("Executor initialized")
	return &wfExecutor
}

// checkErr is a convenience function to panic upon error
func checkErr(err error) {
	if err != nil {
		util.WriteTeriminateMessage(err.Error())
		panic(err.Error())
	}
}
