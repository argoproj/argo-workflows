package commands

import (
	"os"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/argoproj/argo-workflows/v4/cmd/argoexec/commands/artifact"
	cmdutil "github.com/argoproj/argo-workflows/v4/util/cmd"
	kubecli "github.com/argoproj/argo-workflows/v4/util/kube/cli"
	"github.com/argoproj/argo-workflows/v4/util/logging"
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
			ctx = logging.WithLogger(ctx, logger.WithField("argo", true))
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
	command.AddCommand(NewArtifactPluginInitCommand())
	command.AddCommand(NewArtifactPluginSidecarCommand())
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
	ctx = logging.WithLogger(ctx, logger.WithField("argo", true))
	command.SetContext(ctx)

	return &command
}
