package commands

import (
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(artifactsCmd)
	artifactsCmd.AddCommand(artifactsLoadCmd)
}

var artifactsCmd = &cobra.Command{
	Use:   "artifacts",
	Short: "Artifacts commands",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}

var artifactsLoadCmd = &cobra.Command{
	Use:   "load ARTIFACTS_JSON",
	Short: "Load artifacts according to a json specification",
	Run:   loadArtifacts,
}

func loadArtifacts(cmd *cobra.Command, args []string) {

}
