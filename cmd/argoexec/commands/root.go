package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/cmd/argoexec/commands/artifact"
	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/util"
	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	kubecli "github.com/argoproj/argo-workflows/v3/util/kube/cli"
	"github.com/argoproj/argo-workflows/v3/util/logging"
	"github.com/argoproj/argo-workflows/v3/util/logs"
	"github.com/argoproj/argo-workflows/v3/workflow/common"
	"github.com/argoproj/argo-workflows/v3/workflow/executor"
	"github.com/argoproj/argo-workflows/v3/workflow/executor/emissary"
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

func initConfig() {
	cmdutil.SetGLogLevel(glogLevel)
}

func NewRootCommand() *cobra.Command {
	command := cobra.Command{
		Use:   CLIName,
		Short: "argoexec is the executor sidecar to workflow containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
			ctx, logger, err := cmdutil.CmdContextWithLogger(cmd, logLevel, logFormat)
			if err != nil {
				logging.InitLogger().WithError(err).WithFatal().Error(cmd.Context(), "Failed to create argoexec pre-run logger")
				os.Exit(1)
			}

			// Required: argo=true field for test filtering compatibility
			logger = logger.WithField("argo", true)
			ctx = logging.WithLogger(ctx, logger)
			cmd.SetContext(ctx)

			// Disable printing of usage string on errors, except for argument validation errors
			// (i.e. when the "Args" function returns an error).
			//
			// This is set here instead of directly in "command" because Cobra
			// executes PersistentPreRun after performing argument validation:
			// https://github.com/spf13/cobra/blob/3a5efaede9d389703a792e2f7bfe3a64bc82ced9/command.go#L939-L957
			cmd.SilenceUsage = true
		},
	}
	command.AddCommand(NewAgentCommand())
	command.AddCommand(NewEmissaryCommand())
	command.AddCommand(NewInitCommand())
	command.AddCommand(NewKillCommand())
	command.AddCommand(NewResourceCommand())
	command.AddCommand(NewWaitCommand())
	command.AddCommand(NewDataCommand())
	command.AddCommand(cmdutil.NewVersionCmd(CLIName))
	command.AddCommand(artifact.NewArtifactCommand())

	clientConfig = kubecli.AddKubectlFlagsToCmd(&command)
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")

	ctx, logger, err := cmdutil.CmdContextWithLogger(&command, logLevel, logFormat)
	if err != nil {
		logging.InitLogger().WithError(err).WithFatal().Error(command.Context(), "Failed to create argoexec logger")
		os.Exit(1)
	}

	// Required: argo=true field for test filtering compatibility
	logger = logger.WithField("argo", true)
	ctx = logging.WithLogger(ctx, logger)
	command.SetContext(ctx)

	return &command
}

// nolint: contextcheck
func initExecutor(ctx context.Context) *executor.WorkflowExecutor {
	version := argo.GetVersion()
	logger := logging.RequireLoggerFromContext(ctx)
	logger.WithFields(version.Fields()).Info(ctx, "Starting Workflow Executor")
	config, err := clientConfig.ClientConfig()
	checkErr(err)
	config = restclient.AddUserAgent(config, fmt.Sprintf("argo-workflows/%s argo-executor", version.Version))

	// nolint:contextcheck
	bgCtx := logger.NewBackgroundContext()
	logs.AddK8SLogTransportWrapper(bgCtx, config) // lets log all request as we should typically do < 5 per pod, so this is will show up problems

	namespace, _, err := clientConfig.Namespace()
	checkErr(err)

	clientset, err := kubernetes.NewForConfig(config)
	checkErr(err)

	restClient := clientset.RESTClient()

	podName, ok := os.LookupEnv(common.EnvVarPodName)
	if !ok {
		logger.WithFatal().Error(ctx, fmt.Sprintf("Unable to determine pod name from environment variable %s", common.EnvVarPodName))
		os.Exit(1)
	}

	tmpl := &wfv1.Template{}
	envVarTemplateValue, ok := os.LookupEnv(common.EnvVarTemplate)
	// wait container reads template from the file written by init container, instead of from environment variable.
	if !ok {
		data, err := os.ReadFile(varRunArgo + "/template")
		checkErr(err)
		envVarTemplateValue = string(data)
	} else if envVarTemplateValue == common.EnvVarTemplateOffloaded {
		data, err := os.ReadFile(filepath.Join(common.EnvConfigMountPath, common.EnvVarTemplate))
		checkErr(err)
		envVarTemplateValue = string(data)
	}
	checkErr(json.Unmarshal([]byte(envVarTemplateValue), tmpl))

	includeScriptOutput := os.Getenv(common.EnvVarIncludeScriptOutput) == "true"
	deadline, err := time.Parse(time.RFC3339, os.Getenv(common.EnvVarDeadline))
	checkErr(err)

	// errors ignored because values are set by the controller and checked there.
	annotationPatchTickDuration, _ := time.ParseDuration(os.Getenv(common.EnvVarProgressPatchTickDuration))
	progressFileTickDuration, _ := time.ParseDuration(os.Getenv(common.EnvVarProgressFileTickDuration))

	cre, err := emissary.New()
	checkErr(err)

	wfExecutor := executor.NewExecutor(
		ctx,
		clientset,
		versioned.NewForConfigOrDie(config).ArgoprojV1alpha1().WorkflowTaskResults(namespace),
		restClient,
		podName,
		types.UID(os.Getenv(common.EnvVarPodUID)),
		os.Getenv(common.EnvVarWorkflowName),
		types.UID(os.Getenv(common.EnvVarWorkflowUID)),
		os.Getenv(common.EnvVarNodeID),
		namespace,
		cre,
		*tmpl,
		includeScriptOutput,
		deadline,
		annotationPatchTickDuration,
		progressFileTickDuration,
	)

	logger.
		WithFields(version.Fields()).
		WithField("namespace", namespace).
		WithField("podName", podName).
		WithField("templateName", wfExecutor.Template.Name).
		WithField("includeScriptOutput", includeScriptOutput).
		WithField("deadline", deadline).
		Info(ctx, "Executor initialized")
	return &wfExecutor
}

// checkErr is a convenience function to panic upon error
func checkErr(err error) {
	if err != nil {
		util.WriteTerminateMessage(err.Error())
		panic(err.Error())
	}
}
