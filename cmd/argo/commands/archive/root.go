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
   		argo archive list
   		# Get details of a specific workflow archive:
   		argo archive get uid
   		# Delete a specific workflow archive:
   		argo archive delete uid
   		# List workflow archives by label key:
   		argo archive list-label-key
   		# List workflow archives by label value:
   		argo archive list-label-value
   		# Resubmit a workflow archive:
   		argo archive resubmit uid
		# Retry a workflow archive:
   		argo archive retry uid
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
