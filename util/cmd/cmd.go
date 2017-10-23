// Package cmd provides functionally common to various argo CLIs

package cmd

import (
	"fmt"
	"os"

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

// MustIsDir returns whether or not the given filePath is a directory. Exits if path does not exist
func MustIsDir(filePath string) bool {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return fileInfo.IsDir()
}
