package artifact

import (
	"github.com/spf13/cobra"
)

func NewArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "artifact",
	}
	cmd.AddCommand(NewArtifactDeleteCommand())
	return cmd
}
