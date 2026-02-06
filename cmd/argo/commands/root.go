package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/argoproj/argo-workflows/v3"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/archive"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/auth"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/client"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/clustertemplate"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/cron"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/executorplugin"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/sync"
	"github.com/argoproj/argo-workflows/v3/cmd/argo/commands/template"

	cmdutil "github.com/argoproj/argo-workflows/v3/util/cmd"
	grpcutil "github.com/argoproj/argo-workflows/v3/util/grpc"
	"github.com/argoproj/argo-workflows/v3/util/logging"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

// NewCommand returns a new instance of an argo command
func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   CLIName,
		Short: "argo is the command line interface to Argo",
		Long: `
You can use the CLI in the following modes:

#### Kubernetes API Mode (default)

Requests are sent directly to the Kubernetes API. No Argo Server is needed. Large workflows and the workflow archive are not supported.

Use when you have direct access to the Kubernetes API, and don't need large workflow or workflow archive support.

If you're using instance ID (which is very unlikely), you'll need to set it:

	ARGO_INSTANCEID=your-instanceid

#### Argo Server GRPC Mode

Requests are sent to the Argo Server API via GRPC (using HTTP/2). Large workflows and the workflow archive are supported. Network load-balancers that do not support HTTP/2 are not supported.

Use if you do not have access to the Kubernetes API (e.g. you're in another cluster), and you're running the Argo Server using a network load-balancer that support HTTP/2.

To enable, set ARGO_SERVER:

	ARGO_SERVER=localhost:2746 ;# The format is "host:port" - do not prefix with "http" or "https"

If you're have transport-layer security (TLS) enabled (i.e. you are running "argo server --secure" and therefore has HTTPS):

	ARGO_SECURE=true

If your server is running with self-signed certificates. Do not use in production:

	ARGO_INSECURE_SKIP_VERIFY=true

By default, the CLI uses your KUBECONFIG to determine default for ARGO_TOKEN and ARGO_NAMESPACE. You probably error with "no configuration has been provided". To prevent it:

	KUBECONFIG=/dev/null

You will then need to set:

	ARGO_NAMESPACE=argo

And:

	ARGO_TOKEN='Bearer ******' ;# Should always start with "Bearer " or "Basic ".

#### Argo Server HTTP1 Mode

As per GRPC mode, but uses HTTP. Can be used with ALB that does not support HTTP/2. The command "argo logs --since-time=2020...." will not work (due to time-type).

Use this when your network load-balancer does not support HTTP/2.

Use the same configuration as GRPC mode, but also set:

	ARGO_HTTP1=true

If your server is behind an ingress with a path (running "argo server --base-href /argo" or "ARGO_BASE_HREF=/argo argo server"):

	ARGO_BASE_HREF=/argo
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	command.AddCommand(NewCompletionCommand())
	command.AddCommand(NewConvertCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewLogsCommand())
	command.AddCommand(NewResubmitCommand())
	command.AddCommand(NewResumeCommand())
	command.AddCommand(NewRetryCommand())
	command.AddCommand(NewServerCommand())
	command.AddCommand(NewSubmitCommand())
	command.AddCommand(NewSuspendCommand())
	command.AddCommand(auth.NewAuthCommand())
	command.AddCommand(NewWaitCommand())
	command.AddCommand(NewWatchCommand())
	command.AddCommand(NewCpCommand())
	command.AddCommand(NewStopCommand())
	command.AddCommand(NewNodeCommand())
	command.AddCommand(NewTerminateCommand())
	command.AddCommand(archive.NewArchiveCommand())
	command.AddCommand(NewVersionCommand())
	command.AddCommand(template.NewTemplateCommand())
	command.AddCommand(cron.NewCronWorkflowCommand())
	command.AddCommand(clustertemplate.NewClusterTemplateCommand())
	command.AddCommand(executorplugin.NewRootCommand())
	command.AddCommand(sync.NewSyncCommand())

	client.AddKubectlFlagsToCmd(command)
	client.AddAPIClientFlagsToCmd(command)
	// global log level
	var logLevel string
	var logFormat string
	var glogLevel int
	var verbose bool
	command.PersistentPostRun = func(cmd *cobra.Command, args []string) {
		cmdutil.PrintVersionMismatchWarning(cmd.Context(), argo.GetVersion(), grpcutil.LastSeenServerVersion)
	}
	command.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			logLevel = "debug"
			glogLevel = 6
		}
		ctx, log, err := cmdutil.CmdContextWithLogger(cmd, logLevel, logFormat)
		if err != nil {
			logging.InitLogger().WithError(err).WithFatal().Error(ctx, "Failed to create argo pre-run logger")
			os.Exit(1)
		}

		logging.SetupKlogAdapter(ctx)
		cmdutil.SetGLogLevel(glogLevel)
		command.SetContext(ctx)

		log.WithField("version", argo.GetVersion()).Debug(ctx, "CLI version")

		// Disable printing of usage string on errors, except for argument validation errors
		// (i.e. when the "Args" function returns an error).
		//
		// This is set here instead of directly in "command" because Cobra
		// executes PersistentPreRun after performing argument validation:
		// https://github.com/spf13/cobra/blob/3a5efaede9d389703a792e2f7bfe3a64bc82ced9/command.go#L939-L957
		cmd.SilenceUsage = true
	}
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")
	command.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enabled verbose logging, i.e. --loglevel debug")
	cctx, log, err := cmdutil.CmdContextWithLogger(command, logLevel, logFormat)
	if err != nil {
		logging.InitLogger().WithError(err).WithFatal().Error(cctx, "Failed to create argo logger")
		os.Exit(1)
	}
	command.SetContext(cctx)

	// set-up env vars for the CLI such that ARGO_* env vars can be used instead of flags
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ARGO")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	// bind flags to env vars (https://github.com/spf13/viper/tree/v1.17.0#working-with-flags)
	if err := viper.BindPFlags(command.PersistentFlags()); err != nil {
		log.WithError(err).WithFatal().Error(cctx, "Failed to bind flags to env vars")
	}
	// workaround for handling required flags (https://github.com/spf13/viper/issues/397#issuecomment-544272457)
	command.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			if err := command.PersistentFlags().Set(f.Name, fmt.Sprintf("%v", val)); err != nil {
				log.WithError(err).WithFatal().Error(cctx, "Failed to set flag")
			}
		}
	})

	return command
}
