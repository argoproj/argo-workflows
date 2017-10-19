// Package cmd provides functionally common to various argo CLIs

package cmd

import (
	"fmt"

	"github.com/argoproj/argo"
	"github.com/spf13/cobra"
)

// NewVersionCmd returns a new `version` command to be used as a sub-command to root
func NewVersionCmd(cliName string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: fmt.Sprintf("Print version information"),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s version %s\n", cliName, argo.FullVersion)
		},
	}
}
