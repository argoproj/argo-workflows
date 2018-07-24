package cmd

import (
	"github.com/gobuffalo/packr/builder"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "removes any *-packr.go files",
	Run: func(cmd *cobra.Command, args []string) {
		builder.Clean(input)
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
