package commands

import (
	"github.com/argoproj/argo-workflows/v3/util/cmd"
	"github.com/argoproj/pkg/cli"
	"github.com/spf13/cobra"
)

const (
	CLIName = "cwl2argo"
)

var (
	logLevel  string
	logFormat string
)

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	cmd.SetLogFormatter(logFormat)
	cli.SetLogLevel(logLevel)
}

func NewRootCommand() *cobra.Command {
	command := cobra.Command{
		Use:   CLIName,
		Short: "cwl2argo transpiles cwl to argo yaml",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewTranspileCommand())
	command.PersistentFlags().StringVar(&logLevel, "loglevel", "info", "Set the logging level. One of: debug|info|warn|error")
	command.PersistentFlags().StringVar(&logFormat, "log-format", "text", "The formatter to use for logs. One of: text|json")

	return &command
}
