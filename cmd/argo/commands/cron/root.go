package cron

import (
	"github.com/spf13/cobra"
)

func NewCronWorkflowCommand() *cobra.Command {
	var command = &cobra.Command{
		Use:          "cron",
		Short:        "manage cron workflows",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.HelpFunc()(cmd, args)
			return nil
		},
	}

	command.AddCommand(NewGetCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewSuspendCommand())
	command.AddCommand(NewResumeCommand())

	return command
}
