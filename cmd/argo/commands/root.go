package commands

import (
	"fmt"

	"github.com/argoproj/pkg/cli"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/argoproj/argo"
	"github.com/argoproj/argo/cmd/argo/commands/clustertemplate"

	"github.com/argoproj/argo/cmd/argo/commands/auth"
	"github.com/argoproj/argo/cmd/argo/commands/cron"
	"github.com/argoproj/argo/util/help"

	"github.com/argoproj/argo/cmd/argo/commands/archive"
	"github.com/argoproj/argo/cmd/argo/commands/client"
	"github.com/argoproj/argo/cmd/argo/commands/template"
)

const (
	// CLIName is the name of the CLI
	CLIName = "argo"
)

// NewCommand returns a new instance of an argo command
func NewCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:     CLIName,
		Short:   "argo is the command line interface to Argo",
		Example: fmt.Sprintf(`If you're using the Argo Server (e.g. because you need large workflow support or workflow archive), please read %s.`, help.CLI),
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}

	command.AddCommand(NewCompletionCommand())
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
	command.AddCommand(NewStopCommand())
	command.AddCommand(NewNodeCommand())
	command.AddCommand(NewTerminateCommand())
	command.AddCommand(archive.NewArchiveCommand())
	command.AddCommand(NewVersionCommand())
	command.AddCommand(template.NewTemplateCommand())
	command.AddCommand(cron.NewCronWorkflowCommand())
	command.AddCommand(clustertemplate.NewClusterTemplateCommand())

	client.AddKubectlFlagsToCmd(command)
	client.AddAPIClientFlagsToCmd(command)
	// global log level
	var logLevel string
	var glogLevel int
	var verbose bool
	command.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if verbose {
			logLevel = "debug"
			glogLevel = 6
		}
		cli.SetLogLevel(logLevel)
		cli.SetGLogLevel(glogLevel)
		log.WithField("version", argo.GetVersion()).Debug("CLI version")
	}
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().IntVar(&glogLevel, "gloglevel", 0, "Set the glog logging level")
	command.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enabled verbose logging, i.e. --loglevel debug")

	return command
}
