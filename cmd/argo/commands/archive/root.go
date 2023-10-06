package archive

import (
	"github.com/spf13/cobra"
)

func NewArchiveCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "archive",
		Short: "manage the workflow archive",
		Example: `
  		# List workflow archives:
  		archive list

  		# Get details of a specific workflow archive:
  		archive get [workflow ID]

  		# Delete a specific workflow archive:
  		archive delete [workflow ID]

  		# List workflow archives by label key:
  		archive list-label-key

  		# List workflow archives by label value:
  		archive list-label-value

  		# Resubmit a workflow archive:
  		archive resubmit [workflow ID]

 		 # Retry a workflow archive:
  		archive retry [workflow ID]
  		`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.HelpFunc()(cmd, args)
		},
	}
	command.AddCommand(NewListCommand())
	command.AddCommand(NewGetCommand())
	command.AddCommand(NewDeleteCommand())
	command.AddCommand(NewListLabelKeyCommand())
	command.AddCommand(NewListLabelValueCommand())
	command.AddCommand(NewResubmitCommand())
	command.AddCommand(NewRetryCommand())
	return command
}
