package artifact

import (
	"github.com/argoproj/argo-workflows/v3/util"
	"github.com/spf13/cobra"
)

func NewArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "artifact",
	}
	cmd.AddCommand(NewArtifactDeleteCommand())
	return cmd
}

// checkErr is a convenience function to panic upon error
func checkErr(err error) {
	if err != nil {
		util.WriteTerminateMessage(err.Error())
		panic(err.Error())
	}
}
