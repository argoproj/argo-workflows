package archive

import (
	"fmt"

	"github.com/argoproj/pkg/errors"
	"github.com/spf13/cobra"

	v1 "github.com/argoproj/argo/cmd/argo/commands/client/v1"
)

func NewDeleteCommand() *cobra.Command {
	var command = &cobra.Command{
		Use: "delete UID...",
		Run: func(cmd *cobra.Command, args []string) {
			for _, uid := range args {
				client, err := v1.GetClient()
				errors.CheckError(err)
				err = client.DeleteArchivedWorkflow(uid)
				errors.CheckError(err)
				fmt.Printf("Archived workflow '%s' deleted\n", uid)
			}
		},
	}
	return command
}
