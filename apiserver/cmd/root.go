package cmd

import (
	"github.com/spf13/cobra"
)

// RootCmd is the argo root level command
var RootCmd = &cobra.Command{
	Use:   "argo-apiserver",
	Short: "Argo API Server",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.HelpFunc()(cmd, args)
	},
}
