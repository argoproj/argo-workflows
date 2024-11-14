package cron

import (
	"github.com/spf13/cobra"
)

func NewCronWorkflowCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "cron",
		Short: "manage cron workflows",
		Long:  `NextScheduledRun assumes that the workflow-controller uses UTC as its timezone`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(NewGetCommand())
	command.AddCommand(NewListCommand())
	command.AddCommand(NewCreateCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewLintCommand())
	command.AddCommand(NewSuspendCommand())
	command.AddCommand(NewResumeCommand())
	command.AddCommand(NewUpdateCommand())

	return command
}
